// SPDX-FileCopyrightText: 2021 Michael Seplowitz
// SPDX-License-Identifier: MIT

package git

import (
	"fmt"
	"os"
	"path/filepath"
)

func FindReposInDir(root string) ([]string, error) {
	isRepoRoot, err := IsLocalRepoRoot(root)
	if err != nil {
		return nil, err
	}
	if isRepoRoot {
		return []string{root}, nil
	}

	var repos []string

	dirQueue := []string{root}

	for len(dirQueue) > 0 {
		dirPath := dirQueue[0]
		dirQueue = dirQueue[1:]

		dir, err := os.Open(dirPath)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", dirPath, err)
		}

		dirInfos, err := dir.Readdir(0)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", dirPath, err)
		}

		for _, fi := range dirInfos {
			if !fi.IsDir() {
				continue
			}

			path := filepath.Join(dirPath, fi.Name())
			isRepoRoot, err := IsLocalRepoRoot(path)
			if err != nil {
				return nil, err
			}

			if !isRepoRoot {
				dirQueue = append(dirQueue, path)
				continue
			}

			repos = append(repos, path)
		}
	}

	return repos, nil
}
