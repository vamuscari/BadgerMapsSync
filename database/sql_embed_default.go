//go:build !embed
// +build !embed

package database

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
)

// IsEmbedded returns true if SQL files are embedded in the binary
func IsEmbedded() bool {
	return false
}

// GetEmbeddedSQL returns an error since SQL files are not embedded
func GetEmbeddedSQL(databaseType, filename string) (string, error) {
	return "", fmt.Errorf("SQL files are not embedded in this build, use LoadSQL instead")
}

// ListEmbeddedSQLFiles returns an error since SQL files are not embedded
func ListEmbeddedSQLFiles(databaseType string) ([]string, error) {
	// Even though files aren't embedded, we can still list the files from the filesystem
	// This provides a consistent API regardless of build mode
	var files []string

	dirPath := filepath.Join(".", databaseType)
	entries, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read SQL directory %s: %w", databaseType, err)
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			files = append(files, entry.Name())
		}
	}

	return files, nil
}
