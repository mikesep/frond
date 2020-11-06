package main

import (
	"fmt"
)

type syncPruneOptions struct {
	rootOpts *rootOptions
}

func (opts *syncPruneOptions) Execute(args []string) error {
	fmt.Println("syncPruneOptions.Execute")

	// TODO don't remove if there are local changes or stashed changes or unpushed branches unless
	// forced

	return nil
}
