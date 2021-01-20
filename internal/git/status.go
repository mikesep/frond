package git

import (
	// "bufio"
	"bytes"
	// "fmt"
	"os/exec"
	"strings"
)

type Status struct {
	BranchHead string

	ChangedOrRenamed bool
	Unmerged         bool
	Untracked        bool
	Ignored          bool
}

func getStatus(path string) (Status, error) {
	var status Status

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
