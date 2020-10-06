package main

import (
	"errors"
	"os"

	"github.com/jessevdk/go-flags"
)

/*
frond list
frond status
frond stashed

frond branch
  match regexp
frond pull
frond sync (clone and warn/delete)
frond pr
frond do pwd

config
- list orgs
  - exclude a repo

*/

type rootOptions struct {
	Status statusOptions `command:"status"`
	Branch branchOptions `command:"branch"`
	List   listOptions   `command:"list" alias:"ls"`

	Jobs int `short:"j" long:"jobs" value-name:"N" description:"number of jobs to run in parallel"`

	//ConfigPath string `long:"config" env-var:"FROND_CONFIG" default:"frond.json" description:"path to frond config"`
}

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	var opts rootOptions
	opts.Status.rootOpts = &opts
	opts.Branch.rootOpts = &opts

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
