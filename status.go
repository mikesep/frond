package main

import (
	"fmt"
)

type statusOptions struct {
}

func (opts *statusOptions) Execute(args []string) error {
	fmt.Printf("I'm statusOptions.Execute\nargs=%q\nopts=%+v\n", args, opts)
	return nil
}
