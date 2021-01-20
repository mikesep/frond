package sync

// import (
// 	"fmt"
// 	"os"

// 	"github.com/mikesep/frond/internal/github"
// )

type InitOptions struct {
	// Force bool `short:"f" long:"force" description:"Force init if config already exists"`

	// SingleDirForAllRepos bool    `long:"single-dir-for-all-repos" description:"Don't nest repos inside per-org dirs"`
	// OrgPrefixSeparator   *string `long:"org-prefix-separator" description:"Repo dir name = org + SEPARATOR + repo"`

	// Args InitArgs `positional-args:"true" required:"true"`
}

// type InitArgs struct {
// 	Server      string `positional-arg-name:"<server>"`
// 	SearchQuery string `positional-arg-name:"<search query>"`
// }

/*
func (opts *InitOptions) Execute(args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("extra args: %v", args)
	}

	var cfg syncConfig

	switch {
	case opts.Args.Server == "github.com", github.DetectEnterpriseServer(opts.Args.Server):
		cfg.GitHub = &githubConfig{
			// TODO
			// Server:               opts.Args.Server,
			// SearchQuery:          opts.Args.SearchQuery,
			// SingleDirForAllRepos: opts.SingleDirForAllRepos,
			// OrgPrefixSeparator:   opts.OrgPrefixSeparator,
		}
	default:
		return fmt.Errorf("could not determine server type for %q", opts.Args.Server)
	}

	if _, err := os.Stat(syncConfigFile); err == nil && !opts.Force {
		return fmt.Errorf("%s exists -- use --force to override", syncConfigFile)
	}

	return writeSyncConfig(cfg)
}
*/
