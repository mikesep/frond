package sync

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/mikesep/frond/internal/git"
	"github.com/mikesep/frond/internal/github"
)

type InitOptions struct {
	Force bool `short:"f" long:"force" description:"Force init if config already exists."`

	GitHub gitHubInitOptions
}

func (opts *InitOptions) Execute(args []string) error {
	if _, err := os.Stat(syncConfigFile); err == nil && !opts.Force {
		return fmt.Errorf("%s exists -- use --force to override", syncConfigFile)
	}

	if len(args) == 0 {
		return fmt.Errorf("missing arguments")
	}

	var server string

	for _, arg := range args {
		u, err := url.Parse(arg)
		if err != nil {
			return fmt.Errorf("failed to parse %q: %w", arg, err)
		}
		if u.Host == "" {
			return fmt.Errorf("parse %q, but u.Host is empty: %#v", arg, u)
		}

		switch {
		case server == "":
			server = u.Host
		case server != u.Host:
			return fmt.Errorf("cannot init with multiple servers: %q, %q", server, u.Host)
		}
	}

	var cfg syncConfig

	switch {
	case server == "github.com", github.DetectEnterpriseServer(server):
		gitHubCfg, err := opts.GitHub.newConfig(server, args)
		if err != nil {
			return err
		}

		cfg.GitHub = gitHubCfg
	default:
		return fmt.Errorf("unrecognized server type for %q", server)
	}

	if err := writeConfig(cfg); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	// TODO write some helpful comments

	fmt.Printf("Wrote %s\n", syncConfigFile)
	return nil
}

type gitHubInitOptions struct {
	SingleDir              *bool   `long:"single-dir" description:"Use a single dir for all repositories."`
	AccountPrefixSeparator *string `long:"account-prefix-separator" value-name:"SEP" description:"Create dirs named like <account><SEP><repository>"`
}

func (opts *gitHubInitOptions) newConfig(server string, args []string) (*gitHubConfig, error) {
	cfg := &gitHubConfig{
		Server: server,
		Orgs:   map[string]*gitHubConfigCriteriaWithExclusions{},
		Users:  map[string]*gitHubConfigCriteriaWithExclusions{},
	}

	cfg.SingleDirForAllRepos = opts.SingleDir
	cfg.AccountPrefixSeparator = opts.AccountPrefixSeparator

	cred, err := git.FillCredential("https", cfg.Server)
	if err != nil {
		return nil, err
	}

	ghSAT := github.ServerAndToken{
		Server: cfg.Server,
		Token:  cred.Password,
	}

	for _, arg := range args {
		u, err := url.Parse(arg)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %q: %w", arg, err)
		}

		parts := strings.FieldsFunc(u.Path, func(r rune) bool { return r == '/' })
		if len(parts) < 1 || len(parts) > 2 {
			return nil, fmt.Errorf("expected GH/account or GH/account/repository (%q)", args)
		}

		acct, err := ghSAT.GetAccount(context.Background(), parts[0])
		if err != nil {
			return nil, err
		}

		var acctsMap map[string]*gitHubConfigCriteriaWithExclusions
		switch acct.Type {
		case "Organization":
			acctsMap = cfg.Orgs
		case "User":
			acctsMap = cfg.Users
		default:
			return nil, fmt.Errorf("unexpected account type: %+v", acct)
		}

		switch len(parts) {
		case 1:
			if acctsMap[acct.Login] != nil {
				return nil, fmt.Errorf("cannot init both at account and repository level for %q",
					acct.Login)
			}
			acctsMap[acct.Login] = nil
		case 2:
			criteria, ok := acctsMap[acct.Login]
			if ok && criteria == nil {
				return nil, fmt.Errorf("cannot init both at account and repository level for %q",
					acct.Login)
			}
			if !ok {
				criteria = &gitHubConfigCriteriaWithExclusions{}
				acctsMap[acct.Login] = criteria
			}
			criteria.Names = append(criteria.Names, parts[1])
		}
	}

	if len(cfg.Orgs) == 1 && len(cfg.Users) == 0 {
		for name, criteria := range cfg.Orgs {
			cfg.SingleOrg = name
			if criteria != nil {
				cfg.gitHubConfigCriteriaWithExclusions = *criteria
			}
			delete(cfg.Orgs, name)
		}
	} else if len(cfg.Users) == 1 && len(cfg.Orgs) == 0 {
		for name, criteria := range cfg.Users {
			cfg.SingleUser = name
			if criteria != nil {
				cfg.gitHubConfigCriteriaWithExclusions = *criteria
			}
			delete(cfg.Users, name)
		}
	}

	return cfg, nil
}
