// SPDX-FileCopyrightText: 2020 Michael Seplowitz
// SPDX-License-Identifier: MIT

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
