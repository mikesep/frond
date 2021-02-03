package sync

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/mikesep/frond/internal/git"
	giturls "github.com/whilp/git-urls"
	"golang.org/x/term"
)

type Options struct {
	Init InitOptions `command:"init"`

	DryRun    bool `short:"n" long:"dry-run" description:"Print actions instead of doing them."`
	Jobs      *int `short:"j" long:"jobs" value-name:"N" description:"Run up to N actions in parallel. (default: NumCPU)"`
	KeepGoing bool `short:"k" long:"keep-going" description:"Keep going even if an action fails."`
	Prune     bool `short:"p" long:"prune" description:"Remove extra repositories."`

	// TODO --reset to force back to default and fast-forward branches to tracking
}

func (opts *Options) Execute(args []string) error {
	workDir, err := os.Getwd()
	if err != nil {
		return err
	}

	actions, err := buildActionList(workDir, args, os.Stderr)
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
	if term.IsTerminal(int(os.Stdout.Fd())) && !opts.DryRun {
		output = newSerializingReporter(newANSIReporter(os.Stdout, len(actions), maxNameLen))
	} else {
		output = newSerializingReporter(newPlainReporter(os.Stdout, len(actions), maxNameLen))
	}
	output.DrawInitial()

	queue := make(chan syncAction)

	var wg sync.WaitGroup
	workerCtx, cancelWorkers := context.WithCancel(context.Background())
	workers := runtime.NumCPU()
	if opts.Jobs != nil {
		workers = *opts.Jobs
	}

	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go actionWorker(workerCtx, cancelWorkers, &wg, queue, opts, output)
	}

	enqueuedAll := enqueueActions(workerCtx, actions, queue)
	close(queue)
	wg.Wait()

	var note string
	if !enqueuedAll {
		note = "Stopped early due to failures. (Use --keep-going to keep going.)"
	}
	output.Done(note)

	if c := output.NumFailed(); c > 0 {
		return fmt.Errorf("%d FAILED", c)
	}

	return nil
}

func actionWorker(
	ctx context.Context, cancelWorkers context.CancelFunc, wg *sync.WaitGroup,
	queue <-chan syncAction, opts *Options, output reporter,
) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done(): // cancelled
			return

		case action := <-queue:
			if action == nil { // channel closed
				return
			}

			event := action.Do(opts)
			output.HandleEvent(event)
			if event.Type == actionFailed && !opts.KeepGoing {
				cancelWorkers()
				return
			}
		}
	}
}

func enqueueActions(ctx context.Context, actions []syncAction, queue chan<- syncAction) bool {
	enqueuedAll := false

	for i, action := range actions {
		select {
		case <-ctx.Done():
			return enqueuedAll
		case queue <- action:
			if i == len(actions)-1 {
				enqueuedAll = true
			}
		}
	}

	return enqueuedAll
}

//------------------------------------------------------------------------------

type idealRepo struct {
	Path          string
	URL           string
	DefaultBranch string
}

type idealRepoMap map[string]idealRepo    // comparable URL -> repoAtPath
type rejectionReasonMap map[string]string // comparable URL -> rejection reason

func buildActionList(workDir string, cmdArgs []string, console io.Writer) ([]syncAction, error) {
	if !filepath.IsAbs(workDir) {
		panic(fmt.Sprintf("workDir is not absolute: %q", workDir))
	}

	cfgPath, err := findConfigFile(workDir)
	if err != nil {
		if errors.Is(err, errNoConfigFileFound) {
			return nil, fmt.Errorf("%w\nDid you run 'frond sync init' first?", err)
		}
		return nil, err
	}

	cfg, err := parseConfigFromFile(cfgPath)
	if err != nil {
		return nil, err
	}

	syncRoot := filepath.Dir(cfgPath)

	localRepos, err := findRelativeLocalRepos(syncRoot, workDir, cmdArgs, console)
	if err != nil {
		return nil, err
	}
	// fmt.Printf("DEBUG: localRepos:\n")
	// for i, r := range localRepos {
	// 	fmt.Printf("%d: %v\n", i, r)
	// }

	idealRepos := idealRepoMap{}
	rejectionReasons := rejectionReasonMap{}

	if cfg.GitHub != nil {
		idealRepos, rejectionReasons, err = findGitHubRepos(
			syncRoot, workDir, cmdArgs, cfg.GitHub, console)
		if err != nil {
			return nil, err
		}
	}
	// fmt.Printf("DEBUG: idealRepos:\n")
	// for k, v := range idealRepos {
	// 	fmt.Printf("%v: %v\n", k, v)
	// }

	var actions []syncAction

	for _, r := range localRepos {
		action, err := matchRepoToAction(r, idealRepos, rejectionReasons)
		if err != nil {
			return nil, err
		}

		actions = append(actions, action)
	}

	for _, rap := range idealRepos {
		actions = append(actions, actionCloneRepo{
			URL:  rap.URL,
			Path: rap.Path,
		})
	}

	return actions, nil
}

func findRelativeLocalRepos(syncRoot, workDir string, cmdArgs []string, console io.Writer,
) ([]string, error) {
	if !filepath.IsAbs(syncRoot) {
		panic(fmt.Sprintf("syncRoot is not an absolute path: %q", syncRoot))
	}
	if !filepath.IsAbs(workDir) {
		panic(fmt.Sprintf("workDir is not an absolute path: %q", workDir))
	}

	var absTargets []string
	for _, arg := range cmdArgs {
		argPath := arg
		if !filepath.IsAbs(arg) {
			argPath = filepath.Join(workDir, arg)
		}

		relToRoot, err := filepath.Rel(syncRoot, argPath)
		if err != nil {
			return nil, err
		}

		if strings.HasPrefix(relToRoot, "..") {
			return nil, fmt.Errorf("arg %q points outside sync root %q", arg, syncRoot)
		}

		abs, err := filepath.Abs(argPath)
		if err != nil {
			return nil, err
		}

		absTargets = append(absTargets, abs)
	}
	if len(absTargets) == 0 {
		absTargets = []string{syncRoot}
	}

	// De-duplicate repos because relTargets could overlap
	repoSet := map[string]bool{}

	fmt.Fprintf(console, "Finding local repositories... ")
	for _, t := range absTargets {
		if _, err := os.Stat(t); os.IsNotExist(err) {
			continue
		}

		repos, err := git.FindReposInDir(t)
		if err != nil {
			fmt.Fprintf(console, "FAILED!\n")
			return nil, err
		}

		for _, r := range repos {
			rel, err := filepath.Rel(workDir, r)
			if err != nil {
				return nil, err
			}
			repoSet[rel] = true
		}
	}
	fmt.Fprintf(console, "done.\n")

	repos := make([]string, 0, len(repoSet))
	for r := range repoSet {
		repos = append(repos, r)
	}
	sort.Strings(repos)

	return repos, nil
}

// This removes repos from idealRepos as they're matched!
func matchRepoToAction(repoPath string, idealRepos idealRepoMap, rejectionReasons rejectionReasonMap,
) (syncAction, error) {
	localRepo := git.LocalRepo{Root: repoPath}

	remotes, err := localRepo.Remotes()
	if err != nil {
		return nil, err
	}

	type remoteMatch struct {
		RemoteName    string
		ComparableURL string
	}

	var remoteMatches []remoteMatch
	var matchedRejectedURL, matchedRejectionReason string

	for remoteName, remote := range remotes {
		compURL, err := comparableRepoURL(remote.FetchURL)
		if err != nil {
			return nil, err
		}
		if _, ok := idealRepos[compURL]; ok {
			matchedRejectedURL = ""
			matchedRejectionReason = ""

			remoteMatches = append(remoteMatches, remoteMatch{
				RemoteName:    remoteName,
				ComparableURL: compURL,
			})
			continue
		}

		if reason, ok := rejectionReasons[compURL]; ok {
			matchedRejectedURL = compURL
			matchedRejectionReason = reason
		}
	}

	switch len(remoteMatches) {
	case 0:
		if matchedRejectedURL != "" {
			return actionRemoveRepo{
				Path:   localRepo.Root,
				Reason: matchedRejectionReason,
			}, nil
		}

		return actionRemoveRepo{
			Path:   localRepo.Root,
			Reason: "did not match any remote repo URL",
		}, nil

	case 1:
		matchedRemote := remoteMatches[0]

		ideal := idealRepos[matchedRemote.ComparableURL]
		delete(idealRepos, matchedRemote.ComparableURL)

		defaultTrackingBranch := fmt.Sprintf("%s/%s", matchedRemote.RemoteName, ideal.DefaultBranch)

		if localRepo.Root != ideal.Path {
			return actionMoveAndSyncRepo{
				OrigPath:              localRepo.Root,
				DestPath:              ideal.Path,
				DefaultTrackingBranch: defaultTrackingBranch,
			}, nil
		}

		return actionSyncRepo{
			Path:                  localRepo.Root,
			DefaultTrackingBranch: defaultTrackingBranch,
		}, nil

	default:
		return nil, fmt.Errorf("%s matched with more than one URL: %v",
			localRepo.Root, remoteMatches)
	}
}

//------------------------------------------------------------------------------

type syncAction interface {
	Name() string
	Do(opts *Options) actionEvent
}

type actionCloneRepo struct {
	URL  string
	Path string
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
			Type:    actionCloned,
			Name:    a.Path,
			Message: fmt.Sprintf("would clone from %s", a.URL),
		}
	}

	return cloneRepo(a.URL, a.Path)
}

type actionMoveAndSyncRepo struct {
	OrigPath              string
	DestPath              string
	DefaultTrackingBranch string
}

func (a actionMoveAndSyncRepo) Name() string {
	return a.DestPath
}

func (a actionMoveAndSyncRepo) Do(opts *Options) actionEvent {
	if _, err := os.Stat(a.DestPath); err != nil || !os.IsNotExist(err) {
		return actionEvent{
			Type:    actionFailed,
			Name:    a.OrigPath,
			Message: fmt.Sprintf("would move to %s, but it already exists", a.DestPath),
		}
	}

	if opts.DryRun {
		return actionEvent{
			Type:    actionUpdated,
			Name:    a.DestPath,
			Message: fmt.Sprintf("would move to %s and sync", a.DestPath),
		}
	}

	if err := os.Rename(a.OrigPath, a.DestPath); err != nil {
		return actionEvent{
			Type:    actionFailed,
			Name:    a.OrigPath,
			Message: err.Error(),
		}
	}

	return syncRepo(a.DestPath, a.DefaultTrackingBranch)
}

type actionRemoveRepo struct {
	Path   string
	Reason string
}

func (a actionRemoveRepo) Name() string {
	return a.Path
}

func (a actionRemoveRepo) Do(opts *Options) actionEvent {
	if opts.DryRun {
		if opts.Prune {
			return actionEvent{
				Type:    actionRemoved,
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

	// TODO check for unpushed work on branches that don't line up with their tracking branches

	err := os.RemoveAll(a.Path)
	if err != nil {
		return actionEvent{
			Type:    actionFailed,
			Name:    a.Path,
			Message: err.Error(),
		}
	}

	return actionEvent{
		Type:    actionRemoved,
		Name:    a.Path,
		Message: "removed",
	}
}

type actionSyncRepo struct {
	Path                  string
	DefaultTrackingBranch string
}

func (a actionSyncRepo) Name() string {
	return a.Path
}

func (a actionSyncRepo) Do(opts *Options) actionEvent {
	if opts.DryRun {
		return actionEvent{
			Type:    actionUpdated,
			Name:    a.Path,
			Message: "would sync",
		}
	}

	return syncRepo(a.Path, a.DefaultTrackingBranch)
}

//------------------------------------------------------------------------------

func comparableRepoURL(rawURL string) (string, error) {
	u, err := giturls.Parse(rawURL)
	if err != nil {
		return "", err
	}

	return path.Join(u.Host, strings.TrimSuffix(u.Path, ".git")), nil
}
