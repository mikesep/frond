package main

import (
	"fmt"
	"os"
	"sort"

	"github.com/mikesep/frond/internal/git"
	"github.com/mikesep/frond/internal/github"
)

type syncOptions struct {
	rootOpts *rootOptions

	Init  syncInitOptions  `command:"init"`
	Prune syncPruneOptions `command:"prune"`
}

func (opts *syncOptions) setRootOpts(rootOpts *rootOptions) {
	opts.rootOpts = rootOpts
	opts.Init.rootOpts = rootOpts
}

func (opts *syncOptions) Execute(args []string) error {
	cfg, err := readSyncConfig()
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w\nDid you run 'frond sync init' first?", err)
		}
		return err
	}

	if cfg.GitHub != nil {
		return syncGitHub(*cfg.GitHub)
	}

	return fmt.Errorf("unknown sync type")
}

func syncGitHub(cfg syncConfigGitHub) error {
	cred, err := git.FillCredential("https", cfg.Server)
	if err != nil {
		return err
	}

	ghSAT := github.ServerAndToken{
		Server: cfg.Server,
		Token:  cred.Password,
	}

	localRepos, err := findRepos(".")
	if err != nil {
		return err
	}
	fmt.Printf("DEBUG: local repos: %d\n", len(localRepos))

	remoteRepos, err := ghSAT.SearchForRepositories(cfg.SearchQuery)
	if err != nil {
		return err
	}
	fmt.Printf("DEBUG: remote repos in %s %q: %d\n", cfg.Server, cfg.SearchQuery, len(remoteRepos))

	sort.Strings(localRepos)
	sort.Strings(remoteRepos)

	matchingRepos, missingRepos, extraRepos := diffLocalAndRemoteRepos(localRepos, remoteRepos)

	if len(extraRepos) > 0 {
		repoOrRepos := "repo"
		if len(extraRepos) > 1 {
			repoOrRepos = "repos"
		}
		fmt.Printf("Found %d local %s that didn't match the search.\n", len(extraRepos), repoOrRepos)
		fmt.Printf("To remove the extra %s, run 'frond sync prune' or 'frond sync --prune'.\n",
			repoOrRepos)
	}

	return nil
}

// The slices must be sorted!
func diffLocalAndRemoteRepos(localRepos, remoteRepos []string) (
	matching, missing, extra []string,
) {
	if !sort.StringsAreSorted(localRepos) {
		panic("localRepos slice is not sorted")
	}
	if !sort.StringsAreSorted(remoteRepos) {
		panic("remoteRepos slice is not sorted")
	}

	var localIndex, remoteIndex int

	for localIndex < len(localRepos) && remoteIndex < len(remoteRepos) {
		var local *string
		if localIndex < len(localRepos) {
			local = &localRepos[localIndex]
		}

		var remote *string
		if remoteIndex < len(remoteRepos) {
			remote = &remoteRepos[remoteIndex]
		}

		// fmt.Printf("local=%q remote=%q\n", local, remote)

		switch {
		case local == nil || remote != nil && *remote < *local:
			// remote repo that is missing
			missing = append(missing, *remote)
			remoteIndex++

		case remote == nil || local != nil && *local < *remote:
			// local repo that isn't in remote list
			extra = append(extra, *local)
			localIndex++

		default:
			// local repo that is in remote list
			matching = append(matching, *local)
			localIndex++
			remoteIndex++
		}
	}

	return matching, missing, extra, nil
}
