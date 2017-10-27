package module

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestGetModule(t *testing.T) {
	server := mockRegistry()
	defer server.Close()
	disco := testDisco(server)

	td, err := ioutil.TempDir("", "tf")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(td)
	storage := NewStorage(td, disco, nil)

	// this module exists in a test fixture, and is known by the mockRegistry
	// relative to our cwd.
	err = storage.GetModule(filepath.Join(td, "foo"), "registry/local/sub")
	if err != nil {
		t.Fatal(err)
	}

	// list everything to make sure nothing else got unpacked in here
	ls, err := ioutil.ReadDir(td)
	if err != nil {
		t.Fatal(err)
	}

	var names []string
	for _, info := range ls {
		names = append(names, info.Name())
	}

	if !(len(names) == 1 && names[0] == "foo") {
		t.Fatalf("expected only directory 'foo', found entries %q", names)
	}

	_, err = os.Stat(filepath.Join(td, "foo", "main.tf"))
	if err != nil {
		t.Fatal(err)
	}

}
