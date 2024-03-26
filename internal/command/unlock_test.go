// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"os"
	"testing"

	"github.com/hashicorp/cli"

	"github.com/hashicorp/terraform/internal/backend/remote-state/inmem"
	"github.com/hashicorp/terraform/internal/command/workdir"
)

// Since we can't unlock a local state file, just test that calling unlock
// doesn't fail.
func TestUnlock(t *testing.T) {
	td := t.TempDir()
	os.MkdirAll(td, 0755)
	defer testChdir(t, td)()

	// Write the legacy state
	statePath := DefaultStateFilename
	{
		emptyStateFile := workdir.NewBackendStateFile()
		emptyStateFileRaw, err := workdir.EncodeBackendStateFile(emptyStateFile)
		if err != nil {
			t.Fatal(err)
		}
		err = os.WriteFile(statePath, emptyStateFileRaw, os.ModePerm)
		if err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	p := testProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &UnlockCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{
		"-force",
		"LOCK_ID",
	}

	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: %d\n%s\n%s", code, ui.OutputWriter.String(), ui.ErrorWriter.String())
	}

	// make sure we don't crash with arguments in the wrong order
	args = []string{
		"LOCK_ID",
		"-force",
	}

	if code := c.Run(args); code != cli.RunResultHelp {
		t.Fatalf("bad: %d\n%s\n%s", code, ui.OutputWriter.String(), ui.ErrorWriter.String())
	}
}

// Newly configured backend
func TestUnlock_inmemBackend(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("backend-inmem-locked"), td)
	defer testChdir(t, td)()
	defer inmem.Reset()

	// init backend
	ui := new(cli.MockUi)
	view, _ := testView(t)
	ci := &InitCommand{
		Meta: Meta{
			Ui:   ui,
			View: view,
		},
	}
	if code := ci.Run(nil); code != 0 {
		t.Fatalf("bad: %d\n%s", code, ui.ErrorWriter)
	}

	ui = new(cli.MockUi)
	c := &UnlockCommand{
		Meta: Meta{
			Ui:   ui,
			View: view,
		},
	}

	// run with the incorrect lock ID
	args := []string{
		"-force",
		"LOCK_ID",
	}

	if code := c.Run(args); code == 0 {
		t.Fatalf("bad: %d\n%s\n%s", code, ui.OutputWriter.String(), ui.ErrorWriter.String())
	}

	ui = new(cli.MockUi)
	c = &UnlockCommand{
		Meta: Meta{
			Ui:   ui,
			View: view,
		},
	}

	// lockID set in the test fixture
	args[1] = "2b6a6738-5dd5-50d6-c0ae-f6352977666b"
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n%s\n%s", code, ui.OutputWriter.String(), ui.ErrorWriter.String())
	}

}
