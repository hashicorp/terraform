package module

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/go-getter"
	"github.com/hashicorp/terraform/config"
)

const fixtureDir = "./test-fixtures"

func tempDir(t *testing.T) string {
	t.Helper()
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
	t.Helper()
	c, err := config.LoadDir(filepath.Join(fixtureDir, n))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	return c
}

func testStorage(t *testing.T) getter.Storage {
	t.Helper()
	return &getter.FolderStorage{StorageDir: tempDir(t)}
}
