package state

import (
	"io/ioutil"
	"os"
	"sync"
	"testing"
)

func TestBackupState_locker(t *testing.T) {
	var _ Locker = new(BackupState)
}

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

func TestBackupStateRace(t *testing.T) {
	f, err := ioutil.TempFile("", "tf")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	f.Close()
	defer os.Remove(f.Name())

	ls := testLocalState(t)
	defer os.Remove(ls.Path)
	bs := &BackupState{
		Real: ls,
		Path: f.Name(),
	}

	current := TestStateInitial()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bs.WriteState(current)
			bs.PersistState()
			bs.RefreshState()
		}()
	}

	wg.Wait()
}
