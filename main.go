// SPDX-FileCopyrightText: 2020 Michael Seplowitz
// SPDX-License-Identifier: MIT

package main

import (
	"errors"
	"os"

	"github.com/jessevdk/go-flags"
	"github.com/mikesep/frond/internal/sync"
)

type rootOptions struct {
	Status statusOptions `command:"status"`
	Branch branchOptions `command:"branch"`
	List   listOptions   `command:"list" alias:"ls"`
	Sync   sync.Options  `command:"sync" subcommands-optional:"true"`
}

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	var opts rootOptions

	parser := flags.NewParser(&opts, flags.Default)
	_, err := parser.ParseArgs(args)
	if err != nil {
		var flagErr *flags.Error
		if errors.As(err, &flagErr) {
			if flagErr.Type == flags.ErrHelp {
				return 0
			}
		}
		return 1
	}

	return 0
}
