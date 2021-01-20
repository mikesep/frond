package sync

import (
	"fmt"
	"path/filepath"
)

func matchesAnyFilter(word string, filters []string) bool {
	if len(filters) == 0 {
		return true
	}

	for _, filter := range filters {
		matched, err := filepath.Match(filter, word)
		if err != nil {
			fmt.Printf("WARNING: %v\n", err) // TODO
			return false
		}
		if matched {
			return true
		}
	}

	return false
}

func anyWordMatchesAnyFilter(words []string, filters []string) bool {
	if len(filters) == 0 {
		return true
	}

	for _, word := range words {
		if matchesAnyFilter(word, filters) {
			return true
		}
	}

	return false
}
