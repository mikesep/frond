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
	root string
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

// TODO needed?
// returns "" when detached
func (repo *LocalRepo) CurrentBranch() (string, error) {
	// if repo.currentBranch == nil {
	cmd := exec.Command("git", "branch", "--show-current")
	cmd.Dir = repo.root
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
	// branch := strings.TrimSpace(string(out))
	// repo.currentBranch = &branch
	// } else {
	// fmt.Printf("===> CurrentBranch cached for %s\n", repo.root)
	// }

	// return *repo.currentBranch, nil
}

// TODO needed?
// func (repo *LocalRepo) CurrentUpstreamRemoteName() (string, error) {
// 	if repo.currentUpstreamRemoteName == nil {
// 		branch, err := repo.CurrentBranch()
// 		if err != nil {
// 			return "", err
// 		}

// 		name, err := getUpstreamRemoteName(repo.root, branch)
// 		if err != nil {
// 			return "", err
// 		}

// 		repo.currentUpstreamRemoteName = &name
// 	} else {
// 		fmt.Printf("===> CurrentUpstreamRemoteName cached for %s\n", repo.root)
// 	}

// 	return *repo.currentUpstreamRemoteName, nil
// }

// func getUpstreamRemoteName(dir, branch string) (string, error) {
// 	cmd := exec.Command("git", "branch", "--list", "--format", "%(upstream:remotename)", branch)
// 	cmd.Dir = dir
// 	fmt.Printf("%+v\n", cmd.Args)
// 	out, err := cmd.CombinedOutput()
// 	if err != nil {
// 		return "", err
// 	}

// 	return strings.TrimSpace(string(out)), nil
// }

type LocalRepoRemotes map[string]*LocalRepoRemoteURLs // name -> URLs

type LocalRepoRemoteURLs struct {
	FetchURL string
	PushURL  string
}

func (repo *LocalRepo) Remotes() (LocalRepoRemotes, error) {
	// if repo.allRemotes == nil {
	cmd := exec.Command("git", "remote", "--verbose")
	cmd.Dir = repo.root
	// fmt.Printf("DEBUG: %v\n", cmd.Args)
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
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return remotes, nil
	// repo.allRemotes = &remotes
	// } else {
	// 	fmt.Printf("===> Remotes cached for %s\n", repo.root)
	// }

	// return *repo.allRemotes, nil
}

type LocalRepoBranches map[string]LocalRepoBranch // branch name -> details

type LocalRepoBranch struct {
	UpstreamBranch string
	UpstreamTrack  string
}

func (repo *LocalRepo) LocalBranches() (branches LocalRepoBranches, current string, err error) {
	cmd := exec.Command("git", "branch", "--list",
		"--format", "%(refname:short)\t%(HEAD)\t%(upstream:short)\t%(upstream:track,nobracket)")
	cmd.Dir = repo.root
	// fmt.Printf("DEBUG: %v\n", cmd.Args)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, "", err
	}

	branches = map[string]LocalRepoBranch{}

	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), "\t")

		branches[fields[0]] = LocalRepoBranch{
			UpstreamBranch: fields[2],
			UpstreamTrack:  fields[3],
		}

		if fields[1] == "*" {
			current = fields[0]
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, "", err
	}

	return branches, current, nil
}

func (repo *LocalRepo) DeleteBranch(branch string, force bool) error {
	cmd := exec.Command("git", "branch", "--delete")
	if force {
		cmd.Args = append(cmd.Args, "--force")
	}
	cmd.Args = append(cmd.Args, branch)
	// fmt.Printf("DEBUG: %v\n", cmd.Args)

	cmd.Dir = repo.root
	_, err := cmd.CombinedOutput()
	return err
}

func (repo *LocalRepo) FastForwardMerge() error {
	cmd := exec.Command("git", "merge", "--ff-only")
	// fmt.Printf("DEBUG: %v\n", cmd.Args)
	cmd.Dir = repo.root
	_, err := cmd.CombinedOutput()
	return err
}

func (repo *LocalRepo) FetchAllAndPrune() (updated bool, err error) {
	cmd := exec.Command("git", "fetch", "--prune", "--all")
	// fmt.Printf("DEBUG: %v\n", cmd.Args)
	cmd.Dir = repo.root
	var out []byte
	out, err = cmd.CombinedOutput()
	if err != nil {
		return false, err
	}

	fetchingPrefix := []byte("Fetching ")
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		if !bytes.HasPrefix(scanner.Bytes(), fetchingPrefix) {
			return true, nil
		}
	}

	return false, err
}

func (repo *LocalRepo) ResetBranch(branch, startPoint string) error {
	cmd := exec.Command("git", "branch", "--force", branch, startPoint)
	// fmt.Printf("DEBUG: %v\n", cmd.Args)
	cmd.Dir = repo.root
	_, err := cmd.CombinedOutput()
	return err
}

func (repo *LocalRepo) SwitchToExistingBranch(branch string) error {
	cmd := exec.Command("git", "switch", "--no-guess", branch)
	// fmt.Printf("DEBUG: %v\n", cmd.Args)
	cmd.Dir = repo.root
	_, err := cmd.CombinedOutput()
	return err
}

func (repo *LocalRepo) SwitchToNewTrackingBranch(upstream string) error {
	cmd := exec.Command("git", "switch", "--track", upstream)
	// fmt.Printf("DEBUG: %v\n", cmd.Args)
	cmd.Dir = repo.root
	_, err := cmd.CombinedOutput()
	return err
}
