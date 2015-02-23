package state

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestBackupState(t *testing.T) {
	f, err := ioutil.TempFile("", "tf")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	f.Close()
	defer os.Remove(f.Name())

	ls := testLocalState(t)
	defer os.Remove(ls.Path)
	TestState(t, &BackupState{
		Real: ls,
		Path: f.Name(),
	})

	if fi, err := os.Stat(f.Name()); err != nil {
		t.Fatalf("err: %s", err)
	} else if fi.Size() == 0 {
		t.Fatalf("bad: %d", fi.Size())
	}
}
