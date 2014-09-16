package module

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileGetter_impl(t *testing.T) {
	var _ Getter = new(FileGetter)
}

func TestFileGetter(t *testing.T) {
	g := new(FileGetter)
	dst := tempDir(t)

	// With a dir that doesn't exist
	if err := g.Get(dst, testModuleURL("basic")); err != nil {
		t.Fatalf("err: %s", err)
	}

	// Verify the destination folder is a symlink
	fi, err := os.Lstat(dst)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Fatal("destination is not a symlink")
	}

	// Verify the main file exists
	mainPath := filepath.Join(dst, "main.tf")
	if _, err := os.Stat(mainPath); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestFileGetter_sourceFile(t *testing.T) {
	g := new(FileGetter)
	dst := tempDir(t)

	// With a source URL that is a path to a file
	u := testModuleURL("basic")
	u.Path += "/main.tf"
	if err := g.Get(dst, u); err == nil {
		t.Fatal("should error")
	}
}

func TestFileGetter_sourceNoExist(t *testing.T) {
	g := new(FileGetter)
	dst := tempDir(t)

	// With a source URL that doesn't exist
	u := testModuleURL("basic")
	u.Path += "/main"
	if err := g.Get(dst, u); err == nil {
		t.Fatal("should error")
	}
}

func TestFileGetter_dir(t *testing.T) {
	g := new(FileGetter)
	dst := tempDir(t)

	if err := os.MkdirAll(dst, 0755); err != nil {
		t.Fatalf("err: %s", err)
	}

	// With a dir that exists that isn't a symlink
	if err := g.Get(dst, testModuleURL("basic")); err == nil {
		t.Fatal("should error")
	}
}

func TestFileGetter_dirSymlink(t *testing.T) {
	g := new(FileGetter)
	dst := tempDir(t)
	dst2 := tempDir(t)

	// Make parents
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		t.Fatalf("err: %s", err)
	}
	if err := os.MkdirAll(dst2, 0755); err != nil {
		t.Fatalf("err: %s", err)
	}

	// Make a symlink
	if err := os.Symlink(dst2, dst); err != nil {
		t.Fatalf("err: %s")
	}

	// With a dir that exists that isn't a symlink
	if err := g.Get(dst, testModuleURL("basic")); err != nil {
		t.Fatalf("err: %s", err)
	}

	// Verify the main file exists
	mainPath := filepath.Join(dst, "main.tf")
	if _, err := os.Stat(mainPath); err != nil {
		t.Fatalf("err: %s", err)
	}
}
