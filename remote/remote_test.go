package remote

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureDirectory(t *testing.T) {
	err := EnsureDirectory()
	if err != nil {
		t.Fatalf("Err: %v", err)
	}

	cwd, _ := os.Getwd()
	path := filepath.Join(cwd, LocalDirectory)

	_, err = os.Stat(path)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
}

func TestHiddenStatePath(t *testing.T) {
	path, err := HiddenStatePath()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	cwd, _ := os.Getwd()
	expect := filepath.Join(cwd, LocalDirectory, HiddenStateFile)

	if path != expect {
		t.Fatalf("bad: %v", path)
	}
}
