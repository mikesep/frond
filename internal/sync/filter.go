package sync

import (
	"path/filepath"
)

func matchesAnyFilter(word string, filters []string) (bool, error) {
	if len(filters) == 0 {
		return true, nil
	}

	for _, filter := range filters {
		matched, err := filepath.Match(filter, word)
		if err != nil {
			return false, err
		}
		if matched {
			return true, nil
		}
	}

	return false, nil
}

func anyWordMatchesAnyFilter(words []string, filters []string) (bool, error) {
	if len(filters) == 0 {
		return true, nil
	}

	for _, word := range words {
		matched, err := matchesAnyFilter(word, filters)
		if err != nil {
			return false, err
		}
		if matched {
			return true, nil
		}
	}

	return false, nil
}
