package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNoViperInCMD(t *testing.T) {
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".go") {
			content, err := os.ReadFile(path)
			if err != nil {
				t.Errorf("could not read file %s: %v", path, err)
			}
			if strings.Contains(string(content), "\"github.com/spf13/viper\"") {
				t.Errorf("found forbidden import of viper in %s", path)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("error walking the path: %v", err)
	}
}
