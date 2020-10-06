package main

import (
	"fmt"
)

type statusOptions struct {
	rootOpts *rootOptions
}

func (opts *statusOptions) Execute(args []string) error {
	fmt.Printf("I'm statusOptions.Execute\nargs=%q\nopts=%+v\n", args, opts.rootOpts)
	return nil
}
