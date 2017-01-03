package variables

import (
	"io/ioutil"
	"path/filepath"
	"testing"
)

func testTempFile(t *testing.T) string {
	return filepath.Join(testTempDir(t), "temp.dat")
}

func testTempDir(t *testing.T) string {
	d, err := ioutil.TempDir("", "tf")
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	return d
}
