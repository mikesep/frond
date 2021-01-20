package git

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// git branch --show-current

// git rev-parse --absolute-git-dir

type LocalRepo struct {
	root                      string
	currentBranch             *string
	currentUpstreamRemoteName *string

	allRemotes *LocalRepoRemotes
}

type LocalRepoRemotes map[string]*LocalRepoRemoteURLs // name -> URLs

type LocalRepoRemoteURLs struct {
	FetchURL string
	PushURL  string
}

var ErrNotRepoRoot = fmt.Errorf("not the repo root")

func NewLocalRepoAtDir(dir string) (*LocalRepo, error) {
	var repo LocalRepo
	repo.root = dir

	cmd := exec.Command("git", "rev-parse", "--absolute-git-dir")
	cmd.Dir = repo.root
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	expected, err := filepath.Abs(filepath.Join(dir, ".git"))
	if err != nil {
		return nil, err
	}

	actual := strings.TrimSpace(string(out))
	if expected != actual {
		return nil, fmt.Errorf("%w: expected=%q actual=%q", ErrNotRepoRoot, expected, actual)
	}

	return &repo, nil
}

func (repo *LocalRepo) Root() string {
	return repo.root
}

// returns "" when detached
func (repo *LocalRepo) CurrentBranch() (string, error) {
	if repo.currentBranch == nil {
		cmd := exec.Command("git", "branch", "--show-current")
		cmd.Dir = repo.root
		out, err := cmd.CombinedOutput()
		if err != nil {
			return "", err
		}

		branch := strings.TrimSpace(string(out))
		repo.currentBranch = &branch
	}

	return *repo.currentBranch, nil
}

func (repo *LocalRepo) CurrentUpstreamRemoteName() (string, error) {
	if repo.currentUpstreamRemoteName == nil {
		branch, err := repo.CurrentBranch()
		if err != nil {
			return "", err
		}

		name, err := getUpstreamRemoteName(repo.root, branch)
		if err != nil {
			return "", err
		}

		repo.currentUpstreamRemoteName = &name
	}

	return *repo.currentUpstreamRemoteName, nil
}

func getUpstreamRemoteName(dir, branch string) (string, error) {
	cmd := exec.Command("git", "branch", "--list", "--format", "%(upstream:remotename)", branch)
	cmd.Dir = dir
	fmt.Printf("%+v\n", cmd.Args)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}

// TODO needed?
// func getRemoteURL(dir, name string) (string, error) {
// 	cmd := exec.Command("git", "remote", "get-url", name)
// 	cmd.Dir = dir
// 	fmt.Printf("%+v\n", cmd.Args)
// 	out, err := cmd.CombinedOutput()
// 	if err != nil {
// 		return "", err
// 	}

// 	return strings.TrimSpace(string(out)), nil
// }

func (repo *LocalRepo) Remotes() (LocalRepoRemotes, error) {
	if repo.allRemotes == nil {
		cmd := exec.Command("git", "remote", "--verbose")
		cmd.Dir = repo.root
		// fmt.Printf("%+v\n", cmd.Args)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return nil, err
		}

		remotes := LocalRepoRemotes{}

		scanner := bufio.NewScanner(bytes.NewReader(out))
		for scanner.Scan() {
			fields := strings.Fields(scanner.Text()) // name\tURL (type)
			name, url, urlType := fields[0], fields[1], strings.Trim(fields[2], "()")

			r, ok := remotes[name]
			if !ok {
				r = &LocalRepoRemoteURLs{}
				remotes[name] = r
			}

			switch urlType {
			case "fetch":
				r.FetchURL = url
			case "push":
				r.PushURL = url
			default:
				panic(fmt.Sprintf("unexpected url type %q in line %q", urlType, scanner.Text()))
			}
		}

		repo.allRemotes = &remotes
	}

	return *repo.allRemotes, nil
}
