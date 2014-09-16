package module

import (
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform/config"
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

func testConfig(t *testing.T, n string) *config.Config {
	c, err := config.LoadDir(filepath.Join(fixtureDir, n))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	return c
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

func testModuleURL(n string) *url.URL {
	u, err := url.Parse(testModule(n))
	if err != nil {
		panic(err)
	}

	return u
}

func testStorage(t *testing.T) Storage {
	return &FolderStorage{StorageDir: tempDir(t)}
}
