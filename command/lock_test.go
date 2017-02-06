package command

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
)

func TestLock(t *testing.T) {
	testData, _ := filepath.Abs("./testdata")

	td := tempDir(t)
	os.MkdirAll(td, 0755)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// Write the legacy state
	statePath := DefaultStateFilename
	{
		f, err := os.Create(statePath)
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		err = terraform.WriteState(testState(), f)
		f.Close()
		if err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	p := testProvider()
	ui := new(cli.MockUi)
	c := &LockCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(p),
			Ui:          ui,
		},
	}

	if code := c.Run(nil); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	unlock, err := testLockState(testData, statePath)
	if err == nil {
		unlock()
		t.Fatal("expected error locking state")
	} else if !strings.Contains(err.Error(), "locked") {
		t.Fatal("does not appear to be a lock error:", err)
	}
}

func TestLock_lockedState(t *testing.T) {
	testData, _ := filepath.Abs("./testdata")

	td := tempDir(t)
	os.MkdirAll(td, 0755)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// Write the legacy state
	statePath := DefaultStateFilename
	{
		f, err := os.Create(statePath)
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		err = terraform.WriteState(testState(), f)
		f.Close()
		if err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	p := testProvider()
	ui := new(cli.MockUi)
	c := &LockCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(p),
			Ui:          ui,
		},
	}

	unlock, err := testLockState(testData, statePath)
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()

	if code := c.Run(nil); code == 0 {
		t.Fatal("expected error when locking a locked state")
	}
}
