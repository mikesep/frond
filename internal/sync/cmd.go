package sync

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"runtime"
	"strings"
	"sync"

	"github.com/mikesep/frond/internal/git"
	giturls "github.com/whilp/git-urls"
)

type Options struct {
	Init InitOptions `command:"init"`
	// Prune PruneOptions `command:"prune"`

	DryRun    bool `short:"n" long:"dry-run" description:"dry run"`
	Prune     bool `short:"p" long:"prune" description:"prune"`
	KeepGoing bool `short:"k" long:"keep-going" description:"Keep going even if a sync fails"`

	Jobs *int `short:"j" long:"jobs" description:"set parallelism"` // TODO

	// TODO jobs parallelism

	// TODO --all    sync from the root (the dir where the sync config was found)
	// TODO sync from the current dir down

	// TODO positional args for what to sync
}

func (opts *Options) Execute(args []string) error {
	cfg, err := parseConfigFromFoundFile()
	if err != nil {
		if errors.Is(err, errNoConfigFileFound) {
			return fmt.Errorf("%w\nDid you run 'frond sync init' first?", err)
		}
		return err
	}

	actions, err := buildActionList(cfg)
	if err != nil {
		return err
	}

	maxNameLen := 0
	for _, a := range actions {
		if ln := len(a.Name()); ln > maxNameLen {
			maxNameLen = ln
		}
	}

	var output reporter

	if opts.DryRun {
		output = newSerializingReporter(newPlainReporter(os.Stderr, len(actions), maxNameLen))
	} else {
		// TODO detect terminal
		output = newSerializingReporter(newANSIReporter(os.Stderr, len(actions), maxNameLen))
	}

	q := make(chan syncAction)

	var wg sync.WaitGroup
	workersCtx, cancelWorkers := context.WithCancel(context.Background())
	workers := runtime.NumCPU()
	if opts.Jobs != nil {
		workers = *opts.Jobs
	}
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for {
				select {
				case <-workersCtx.Done():
					return
				case action := <-q:
					if action == nil {
						return
					}
					event := action.Do(opts)
					output.HandleEvent(event)
					if event.Type == actionFailed {
						if !opts.KeepGoing {
							cancelWorkers()
							return
						}
					}
				}
			}
		}()
	}

	enqueuedAllActions := false
	func() {
		for i, action := range actions {
			select {
			case <-workersCtx.Done():
				return
			case q <- action:
				if i == len(actions)-1 {
					enqueuedAllActions = true
				}
			}
		}
	}()
	close(q)
	wg.Wait()

	var note string
	if !enqueuedAllActions {
		note = "Stopped early due to failures. (Use --keep-going to keep going.)"
	}
	output.Done(note)

	if c := output.NumFailed(); c > 0 {
		return fmt.Errorf("%d FAILED", c)
	}

	return nil
}

//------------------------------------------------------------------------------

func buildActionList(cfg syncConfig) ([]syncAction, error) {
	var actions []syncAction

	localRepos, err := git.FindReposInDir(".")
	if err != nil {
		return nil, err
	}

	idealRepos := map[string]repoAtPath{} // comparable URL -> repoAtPath
	rejectedRepos := map[string]string{}  // comparable URL -> reason for rejection

	if cfg.GitHub != nil {
		idealRepos, rejectedRepos, err = cfg.GitHub.LookForRepos()
		if err != nil {
			return nil, err
		}
	}

	// fmt.Printf("DEBUG: %d local repos:\n", len(localRepos))
	for _, r := range localRepos {
		action, err := matchRepoToAction(r, idealRepos, rejectedRepos)
		if err != nil {
			return nil, err
		}

		actions = append(actions, action)
	}

	for _, rap := range idealRepos {
		actions = append(actions, actionCloneRepo{repoAtPath: rap})
	}

	return actions, nil
}

// This removes repos from idealRepos as they're matched!
func matchRepoToAction(
	r *git.LocalRepo, idealRepos map[string]repoAtPath, rejectedRepos map[string]string,
) (syncAction, error) {
	remotes, err := r.Remotes()
	if err != nil {
		return nil, err
	}

	var matchingIdealURLs []string
	var matchedRejectionURL, matchedRejectionReason string

	for _, remote := range remotes {
		compURL, err := comparableRepoURL(remote.FetchURL)
		if err != nil {
			return nil, err
		}
		if _, ok := idealRepos[compURL]; ok {
			matchedRejectionURL = ""
			matchedRejectionReason = ""

			matchingIdealURLs = append(matchingIdealURLs, compURL)
			continue
		}

		if reason, ok := rejectedRepos[compURL]; ok {
			matchedRejectionURL = compURL
			matchedRejectionReason = reason
		}
	}

	switch len(matchingIdealURLs) {
	case 0:
		if matchedRejectionURL != "" {
			return actionRemoveRepo{
				repoAtPath: repoAtPath{
					URL:  matchedRejectionURL,
					Path: r.Root(),
				},
				Reason: matchedRejectionReason,
			}, nil
		}

		return actionRemoveRepo{
			repoAtPath: repoAtPath{
				URL:  "",
				Path: r.Root(),
			},
			Reason: "did not match any remote repo URL",
		}, nil

	case 1:
		remoteURL := matchingIdealURLs[0]
		ideal := idealRepos[remoteURL]
		// fmt.Printf("DEBUG: %s matched with %s\n", r.Root(), ideal)
		delete(idealRepos, remoteURL)

		if r.Root() != ideal.Path {
			return actionMoveAndSyncRepo{
				repoAtPath: repoAtPath{
					URL:  ideal.URL,
					Path: r.Root(),
				},
				DestPath: ideal.Path,
			}, nil
		}

		return actionSyncRepo{
			repoAtPath: ideal,
		}, nil

	default:
		return nil, fmt.Errorf("%s matched with more than one URL: %v",
			r.Root(), matchingIdealURLs)
	}
}

//------------------------------------------------------------------------------

type syncAction interface {
	Name() string
	Do(opts *Options) actionEvent
}

type actionCloneRepo struct {
	repoAtPath
}

func (a actionCloneRepo) Name() string {
	return a.Path
}

func (a actionCloneRepo) Do(opts *Options) actionEvent {
	if _, err := os.Stat(a.Path); err == nil || !os.IsNotExist(err) {
		return actionEvent{
			Type:    actionFailed,
			Name:    a.Path,
			Message: fmt.Sprintf("would clone from %s, but %s already exists", a.URL, a.Path),
		}
	}

	if opts.DryRun {
		return actionEvent{
			Type:    actionSucceeded,
			Name:    a.Path,
			Message: fmt.Sprintf("would clone from %s", a.URL),
		}
	}

	return cloneRepo(a.repoAtPath)
}

type actionMoveAndSyncRepo struct {
	repoAtPath
	DestPath string
}

func (a actionMoveAndSyncRepo) Name() string {
	return a.DestPath
}

func (a actionMoveAndSyncRepo) Do(opts *Options) actionEvent {
	if _, err := os.Stat(a.DestPath); err != nil || !os.IsNotExist(err) {
		return actionEvent{
			Type:    actionFailed,
			Name:    a.Path,
			Message: fmt.Sprintf("would move to %s, but it already exists", a.DestPath),
		}
	}

	if opts.DryRun {
		return actionEvent{
			Type:    actionSucceeded,
			Name:    a.DestPath,
			Message: fmt.Sprintf("would move to %s and sync from %s", a.DestPath, a.URL),
		}
	}

	if err := os.Rename(a.Path, a.DestPath); err != nil {
		return actionEvent{
			Type:    actionFailed,
			Name:    a.Path,
			Message: err.Error(),
		}
	}

	return syncRepo(repoAtPath{
		URL:  a.URL,
		Path: a.DestPath,
	})
}

type actionRemoveRepo struct {
	repoAtPath
	Reason string
}

func (a actionRemoveRepo) Name() string {
	return a.Path
}

func (a actionRemoveRepo) Do(opts *Options) actionEvent {
	if opts.DryRun {
		if opts.Prune {
			return actionEvent{
				Type:    actionSucceeded,
				Name:    a.Path,
				Message: fmt.Sprintf("would remove: %s", a.Reason),
			}
		}

		return actionEvent{
			Type:    actionIgnored,
			Name:    a.Path,
			Message: fmt.Sprintf("would not remove without --prune: %s", a.Reason),
		}
	}

	if !opts.Prune {
		return actionEvent{
			Type:    actionIgnored,
			Name:    a.Path,
			Message: fmt.Sprintf("keeping extra repo: %s -- use --prune to remove it", a.Reason),
		}
	}

	err := os.RemoveAll(a.Path)
	if err != nil {
		return actionEvent{
			Type:    actionFailed,
			Name:    a.Path,
			Message: err.Error(),
		}
	}

	return actionEvent{
		Type:    actionSucceeded,
		Name:    a.Path,
		Message: "removed",
	}
}

type actionSyncRepo struct {
	repoAtPath
}

func (a actionSyncRepo) Name() string {
	return a.Path
}

func (a actionSyncRepo) Do(opts *Options) actionEvent {
	if opts.DryRun {
		return actionEvent{
			Type:    actionSucceeded,
			Name:    a.Path,
			Message: fmt.Sprintf("would sync from %s", a.URL),
		}
	}

	return syncRepo(a.repoAtPath)
}

type repoAtPath struct {
	Path string
	URL  string
}

//------------------------------------------------------------------------------

func comparableRepoURL(rawURL string) (string, error) {
	u, err := giturls.Parse(rawURL)
	if err != nil {
		return "", err
	}

	return path.Join(u.Host, strings.TrimSuffix(u.Path, ".git")), nil
}
