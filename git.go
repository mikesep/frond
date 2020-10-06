package main

import (
	// "bufio"
	"bytes"
	// "fmt"
	"os/exec"
	"strings"
)

type gitStatus struct {
	BranchHead string

	ChangedOrRenamed bool
	Unmerged         bool
	Untracked        bool
	Ignored          bool
}

func getGitStatus(path string) (gitStatus, error) {
	var status gitStatus

	cmd := exec.Command("git", "status", "--null", "--porcelain=v2",
		"--branch", "--ignored", "--untracked=normal")
	cmd.Dir = path

	output, err := cmd.CombinedOutput()
	if err != nil {
		return status, err
	}

	for _, line := range bytes.Split(output, []byte{0}) {
		if len(line) == 0 {
			continue
		}
		switch line[0] {
		case '#':
			parts := strings.Split(string(line), " ")
			if parts[1] == "branch.head" {
				status.BranchHead = parts[2]
			}

		case '1', // changed
			'2': // renamed or copied
			status.ChangedOrRenamed = true
		case 'u': // unmerged
			status.Unmerged = true
		case '?': // untracked
			status.Untracked = true
		case '!': // ignored
			status.Ignored = true
		}

	}

	return status, nil
}

// git rev-parse refs/stash

// git status --porcelain

// git branch --show-current

// git status

// func gitCurrentBranch(path string) {
// }

// func
