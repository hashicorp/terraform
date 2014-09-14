package module

import (
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

const fixtureDir = "./test-fixtures"

func tempDir(t *testing.T) string {
	dir, err := ioutil.TempDir("", "tf")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if err := os.RemoveAll(dir); err != nil {
		t.Fatalf("err: %s", err)
	}

	return dir
}

func testModule(n string) string {
	p := filepath.Join(fixtureDir, n)
	p, err := filepath.Abs(p)
	if err != nil {
		panic(err)
	}

	var url url.URL
	url.Scheme = "file"
	url.Path = p
	return url.String()
}
