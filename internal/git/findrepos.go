package git

import (
	"os"
	"path/filepath"
)

func FindReposInDir(root string) ([]*LocalRepo, error) {
	var repos []*LocalRepo

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
				repo, err := NewLocalRepoAtDir(fiPath)
				if err == nil {
					repos = append(repos, repo)
				} else {
					dirQueue = append(dirQueue, fiPath)
				}
			}
		}
	}

	return repos, nil
}
