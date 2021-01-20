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

	Jobs int `short:"j" long:"jobs" value-name:"N" description:"Number of jobs to run in parallel"`

	//ConfigPath string `long:"config" env-var:"FROND_CONFIG" default:"frond.json" description:"path to frond config"`
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
