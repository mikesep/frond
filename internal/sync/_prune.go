package sync

import (
	"fmt"
)

type PruneOptions struct {
	DryRun bool `short:"n" long:"dry-run" description:"dry run"`
}

func (opts *PruneOptions) Execute(args []string) error {
	fmt.Println("syncPruneOptions.Execute")

	// TODO don't remove if there are local changes or stashed changes or unpushed branches unless
	// forced

	cfg, err := parseConfigFromFoundFile()
	if err != nil {
		if errors.Is(err, errNoConfigFileFound) {
			return fmt.Errorf("%w\nDid you run 'frond sync init' first?", err)
		}
		return err
	}

	figure out repos

	if opts.DryRun {
		fmt.Printf("would prune %v\n", reposToPrune)
	}

	prune(reposToPrune)

	return nil
}
