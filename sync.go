package main

import (
	"fmt"
	"os"

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

	sat := github.ServerAndToken{
		Server: cfg.Server,
		Token:  cred.Password,
	}

	repos, err := sat.SearchForRepositories(cfg.SearchQuery)

	fmt.Println("SYNC")
	fmt.Printf("ERR:   %v\n", err)
	fmt.Printf("REPOS: %v\n", repos)

	return err
}
