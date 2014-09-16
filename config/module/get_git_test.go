package module

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGitGetter_impl(t *testing.T) {
	var _ Getter = new(GitGetter)
}

func TestGitGetter(t *testing.T) {
	g := new(GitGetter)
	dst := tempDir(t)

	// With a dir that doesn't exist
	if err := g.Get(dst, testModuleURL("basic-git")); err != nil {
		t.Fatalf("err: %s", err)
	}

	// Verify the main file exists
	mainPath := filepath.Join(dst, "main.tf")
	if _, err := os.Stat(mainPath); err != nil {
		t.Fatalf("err: %s", err)
	}
}
