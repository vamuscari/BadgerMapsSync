//go:build embed
// +build embed

package database

import (
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

//go:embed mssql/*.sql postgres/*.sql sqlite3/*.sql
var embeddedSQLFiles embed.FS

// IsEmbedded returns true if SQL files are embedded in the binary
func IsEmbedded() bool {
	return true
}

// GetEmbeddedSQL returns the content of an embedded SQL file
func GetEmbeddedSQL(databaseType, filename string) (string, error) {
	path := filepath.Join(databaseType, filename)
	data, err := embeddedSQLFiles.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read embedded SQL file %s: %w", path, err)
	}
	return string(data), nil
}

// ListEmbeddedSQLFiles returns a list of all embedded SQL files for a given database type
func ListEmbeddedSQLFiles(databaseType string) ([]string, error) {
	var files []string

	entries, err := fs.ReadDir(embeddedSQLFiles, databaseType)
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded SQL directory %s: %w", databaseType, err)
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			files = append(files, entry.Name())
		}
	}

	return files, nil
}
