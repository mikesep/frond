package main

import (
	"fmt"
)

type branchOptions struct {
}

func (opts *branchOptions) Execute(args []string) error {
	fmt.Printf("I'm branchOptions.Execute with args=%q, opts=%v\n", args, opts)
	return nil
}
