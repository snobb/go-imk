package fsops

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

// TODO: Add parameter to configure more filters.
var ignoredDirs = []string{
	"/.git",
	"/.hg",
	"/node_modules",
	"/vendor",
	"/target",
}

// implement Walker interface
var DefaultWalker = WalkerFunc(Walk)

func Walk(path string) ([]string, error) {
	files := make([]string, 0)

	err := filepath.Walk(path, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("unable to access the path > %w", err)
		}

		if info.IsDir() {
			if isIgnored(path) {
				// skipping the dir
				return filepath.SkipDir
			}

			files = append(files, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("unable to walk the path %s > %w", path, err)
	}

	return files, nil
}

func isIgnored(path string) bool {
	for _, sfx := range ignoredDirs {
		if strings.HasSuffix(path, sfx) {
			return true
		}
	}
	return false
}
