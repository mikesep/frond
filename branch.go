package main

import (
	"fmt"
)

type branchOptions struct {
	rootOpts *rootOptions
}

func (opts *branchOptions) Execute(args []string) error {
	fmt.Printf("I'm branchOptions.Execute with args=%q\n", args)
	return nil
}
