package sync

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/mikesep/frond/internal/git"
)

func cloneRepo(url, path string) actionEvent {
	cmd := exec.Command("git", "clone", url, path)
	_, err := cmd.CombinedOutput()

	if err != nil {
		return actionEvent{
			Type:    actionFailed,
			Name:    path,
			Message: err.Error(),
		}
	}

	return actionEvent{
		Type:    actionCloned,
		Name:    path,
		Message: fmt.Sprintf("cloned from %s", url),
	}
}

func syncRepo(repoPath, defaultTrackingBranch string) actionEvent {
	failure := func(err error) actionEvent {
		return actionEvent{
			Type:    actionFailed,
			Name:    repoPath,
			Message: err.Error(),
		}
	}

	repo, err := git.NewLocalRepoAtDir(repoPath)
	if err != nil {
		return failure(err)
	}

	origBranches, currentBranch, err := repo.LocalBranches()
	if err != nil {
		return failure(err)
	}

	updated, err := repo.FetchAllAndPrune()
	if err != nil {
		return failure(err)
	}

	// TODO option to force branches to match tracking branches?
	if !updated {
		return actionEvent{
			Type:    actionUnchanged,
			Name:    repoPath,
			Message: "no updates",
		}
	}

	newBranches, _, err := repo.LocalBranches()
	if err != nil {
		return failure(err)
	}

	if origBranches[currentBranch].UpstreamTrack == "" && // (empty = in sync)
		newBranches[currentBranch].UpstreamTrack == "gone" {

		var branchTrackingRemoteDefault string
		for branchName, branchInfo := range newBranches {
			if branchInfo.UpstreamBranch == defaultTrackingBranch {
				branchTrackingRemoteDefault = branchName
				break
			}
		}

		if branchTrackingRemoteDefault != "" {
			if err := repo.SwitchToExistingBranch(branchTrackingRemoteDefault); err != nil {
				return failure(err)
			}
		} else {
			if err := repo.SwitchToNewTrackingBranch(defaultTrackingBranch); err != nil {
				return failure(err)
			}
		}

		currentBranch, err = repo.CurrentBranch()
		if err != nil {
			return failure(err)
		}
	}

	var caveats []string

	for branch, newInfo := range newBranches {
		switch strings.Fields(newInfo.UpstreamTrack)[0] {
		case "":
			// in sync with upstream

		case "behind":
			if branch == currentBranch {
				if err := repo.FastForwardMerge(); err != nil {
					return failure(err)
				}
			} else {
				if err := repo.ResetBranch(branch, newInfo.UpstreamBranch); err != nil {
					return failure(err)
				}
			}

		case "gone":
			if origBranches[branch].UpstreamTrack == "" { // it was in sync before
				const force = true
				if err := repo.DeleteBranch(branch, force); err != nil {
					return failure(err)
				}
				caveats = append(caveats, fmt.Sprintf("deleted %q", branch))
				break
			}
			fallthrough // wasn't in sync and upstream is gone, so leave it alone

		default:
			caveats = append(caveats,
				fmt.Sprintf("left %q in place since it had unpushed changes", branch))
		}
	}

	return actionEvent{
		Type:    actionUpdated,
		Name:    repoPath,
		Message: "updated",
		Caveats: caveats,
	}
}
