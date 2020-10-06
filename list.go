package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type listOptions struct {
	rootOpts *rootOptions
}

func (opts *listOptions) Execute(args []string) error {
	repos, err := listRepos(args)
	if err != nil {
		return err
	}

	maxRepoLen := maxLength(repos)

	for _, r := range repos {
		st, err := getGitStatus(r)
		if err != nil {
			return err
		}

		if st.ChangedOrRenamed {
			fmt.Printf("c")
		} else {
			fmt.Printf(" ")
		}

		if st.Unmerged {
			fmt.Printf("U")
		} else {
			fmt.Printf(" ")
		}

		if st.Untracked {
			fmt.Printf("?")
		} else {
			fmt.Printf(" ")
		}

		if st.Ignored {
			fmt.Printf("i")
		} else {
			fmt.Printf(" ")
		}

		fmt.Printf(" %s%s %s\n", r, strings.Repeat(" ", maxRepoLen-len(r)), st.BranchHead)
	}

	return nil
}

func maxLength(strs []string) int {
	max := len(strs[0])
	for _, s := range strs {
		if len(s) > max {
			max = len(s)
		}
	}
	return max
}

func listRepos(args []string) ([]string, error) {
	var repos []string

	return repos, filepath.Walk(".", func(org string, info os.FileInfo, err error) error {
		if org == "." || !info.IsDir() {
			return nil
		}

		repoErr := filepath.Walk(org, func(repo string, info os.FileInfo, err error) error {
			if repo == org || !info.IsDir() {
				return nil
			}
			repos = append(repos, repo)
			return filepath.SkipDir
		})

		if repoErr != nil {
			return repoErr
		}

		return filepath.SkipDir
	})
}
