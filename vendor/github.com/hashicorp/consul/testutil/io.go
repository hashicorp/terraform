package testutil

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

// tmpdir is the base directory for all temporary directories
// and files created with TempDir and TempFile. This could be
// achieved by setting a system environment variable but then
// the test execution would depend on whether or not the
// environment variable is set.
//
// On macOS the temp base directory is quite long and that
// triggers a problem with some tests that bind to UNIX sockets
// where the filename seems to be too long. Using a shorter name
// fixes this and makes the paths more readable.
//
// It also provides a single base directory for cleanup.
var tmpdir = "/tmp/consul-test"

func init() {
	if err := os.MkdirAll(tmpdir, 0755); err != nil {
		fmt.Printf("Cannot create %s. Reverting to /tmp\n", tmpdir)
		tmpdir = "/tmp"
	}
}

// TempDir creates a temporary directory within tmpdir
// with the name 'testname-name'. If the directory cannot
// be created t.Fatal is called.
func TempDir(t *testing.T, name string) string {
	if t != nil && t.Name() != "" {
		name = t.Name() + "-" + name
	}
	name = strings.Replace(name, "/", "_", -1)
	d, err := ioutil.TempDir(tmpdir, name)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	return d
}

// TempFile creates a temporary file within tmpdir
// with the name 'testname-name'. If the file cannot
// be created t.Fatal is called. If a temporary directory
// has been created before consider storing the file
// inside this directory to avoid double cleanup.
func TempFile(t *testing.T, name string) *os.File {
	if t != nil && t.Name() != "" {
		name = t.Name() + "-" + name
	}
	f, err := ioutil.TempFile(tmpdir, name)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	return f
}
