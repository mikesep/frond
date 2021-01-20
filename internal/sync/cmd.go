package sync

import (
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/mikesep/frond/internal/git"
	giturls "github.com/whilp/git-urls"
)

type Options struct {
	Init  InitOptions  `command:"init"`
	Prune PruneOptions `command:"prune"`

	DryRun bool `short:"n" long:"dry-run" description:"dry run"`

	// TODO jobs parallelism

}

type repoAtPath struct {
	Path string
	URL  string
}

func (opts *Options) Execute(args []string) error {
	cfg, err := parseConfigFromFoundFile()
	if err != nil {
		if errors.Is(err, errNoConfigFileFound) {
			return fmt.Errorf("%w\nDid you run 'frond sync init' first?", err)
		}
		return err
	}

	idealRepos := map[string]repoAtPath{} // comparable URL -> repoAtPath

	if cfg.GitHub != nil {
		reposAtPaths, err := cfg.GitHub.getReposAtPaths()
		if err != nil {
			return err
		}

		for _, rp := range reposAtPaths {
			compURL, err := comparableRepoURL(rp.URL)
			if err != nil {
				return err
			}
			idealRepos[compURL] = rp
		}
	}

	localRepos, err := git.FindReposInDir(".")
	if err != nil {
		return err
	}

	fmt.Printf("DEBUG: %d local repos:\n", len(localRepos))
	for _, r := range localRepos {
		remotes, err := r.Remotes()
		if err != nil {
			return err
		}
		var matchingURLs []string
		for _, remote := range remotes {
			compURL, err := comparableRepoURL(remote.FetchURL)
			if err != nil {
				return err
			}
			if _, ok := idealRepos[compURL]; ok {
				matchingURLs = append(matchingURLs, compURL)
			}
		}

		switch len(matchingURLs) {
		case 0:
			fmt.Printf("%s did not match any ideal repo\n", r.Root())
		case 1:
			ideal := idealRepos[matchingURLs[0]]
			fmt.Printf("%s matched with %s\n", r.Root(), ideal)
			if r.Root() == ideal.Path {
				fmt.Printf("  it's in the exact right spot!\n")
			} else {
				if _, err := os.Stat(ideal.Path); os.IsNotExist(err) {
					fmt.Printf("  I can move it to %s\n", ideal.Path)
				} else {
					fmt.Printf("  something already exists at %s\n", ideal.Path)
				}
			}
		default:
			fmt.Printf("%s matched with more than one URL: %v\n", r.Root(), matchingURLs)
		}
	}

	// matchingRepos, missingRepos, extraRepos := diffLocalAndRemoteRepos(localRepos, remoteRepos)

	// if len(extraRepos) > 0 {
	// 	repoOrRepos := "repo"
	// 	if len(extraRepos) > 1 {
	// 		repoOrRepos = "repos"
	// 	}
	// 	fmt.Printf("Found %d local %s that didn't match the search.\n", len(extraRepos), repoOrRepos)
	// 	fmt.Printf("To remove the extra %s, run 'frond sync prune' or 'frond sync --prune'.\n",
	// 		repoOrRepos)
	// }

	return nil
}

func comparableRepoURL(rawURL string) (string, error) {
	u, err := giturls.Parse(rawURL)
	if err != nil {
		return "", err
	}

	return path.Join(u.Host, strings.TrimSuffix(u.Path, ".git")), nil
}
