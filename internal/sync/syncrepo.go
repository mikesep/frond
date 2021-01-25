package sync

import (
	"fmt"
	"os/exec"
)

func cloneRepo(repo repoAtPath) actionEvent {
	cmd := exec.Command("git", "clone", repo.URL, repo.Path)
	_, err := cmd.CombinedOutput()

	if err != nil {
		return actionEvent{
			Type:    actionFailed,
			Name:    repo.Path,
			Message: err.Error(),
		}
	}

	return actionEvent{
		Type:    actionSucceeded,
		Name:    repo.Path,
		Message: fmt.Sprintf("cloned from %s", repo.URL),
	}
}

func syncRepo(repo repoAtPath) actionEvent {
	return actionEvent{
		Type:    actionFailed,
		Name:    repo.Path,
		Message: "syncRepo not implemented", // TODO
	}
}
