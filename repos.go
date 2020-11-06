package main

import (
	"os"
	"path/filepath"
)

func findRepos(root string) ([]string, error) {
	var repos []string

	dirQueue := []string{root}

	for len(dirQueue) > 0 {
		dirPath := dirQueue[0]
		dirQueue = dirQueue[1:]

		dir, err := os.Open(dirPath)
		if err != nil {
			return nil, err
		}

		dirInfos, err := dir.Readdir(0)
		if err != nil {
			return nil, err
		}

		for _, fi := range dirInfos {
			if fi.IsDir() {
				fiPath := filepath.Join(dirPath, fi.Name())
				isRepo, err := isGitRepo(fiPath)
				if err != nil {
					return nil, err
				}
				if isRepo {
					repos = append(repos, fiPath)
				} else {
					dirQueue = append(dirQueue, fiPath)
				}
			}
		}
	}

	return repos, nil
}

func isGitRepo(dir string) (bool, error) {
	info, err := os.Stat(filepath.Join(dir, ".git"))
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	return info.IsDir(), nil
}
