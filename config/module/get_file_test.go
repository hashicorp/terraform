package module

import (
	"net/url"
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

func testModuleURL(n string) *url.URL {
	u, err := url.Parse(testModule(n))
	if err != nil {
		panic(err)
	}

	return u
}
