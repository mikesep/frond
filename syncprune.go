package main

import (
	"fmt"
)

type syncPruneOptions struct {
	rootOpts *rootOptions
}

func (opts *syncPruneOptions) Execute(args []string) error {
	fmt.Println("syncPruneOptions.Execute")
	return nil
}
