// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform/internal/backend/remote-state/inmem"
	"github.com/hashicorp/terraform/internal/command/workdir"
)

// Since we can't unlock a local state file, just test that calling unlock
// doesn't fail.
func TestUnlock(t *testing.T) {
	td := tempWorkingDir(t)
	t.Chdir(td.RootModuleDir())

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
	view, done := testView(t)
	c := &UnlockCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
			WorkingDir:       td,
		},
	}

	args := []string{
		"-force",
		"LOCK_ID",
	}

	if code := c.Run(args); code != 1 {
		output := done(t)
		t.Fatalf("bad: %d\n%s\n%s", code, output.Stdout(), output.Stderr())
	}

	// make sure we don't crash with arguments in the wrong order
	args = []string{
		"LOCK_ID",
		"-force",
	}

	if code := c.Run(args); code == 0 {
		output := done(t)
		t.Fatalf("bad: %d\n%s\n%s", code, output.Stdout(), output.Stderr())
	}
}

// Newly configured backend
func TestUnlock_inmemBackend(t *testing.T) {
	td := tempWorkingDirFixture(t, "backend-inmem-locked")
	t.Chdir(td.RootModuleDir())

	defer inmem.Reset()

	// init backend
	view, done := testView(t)
	ci := &InitCommand{
		Meta: Meta{
			View:       view,
			WorkingDir: td,
		},
	}
	if code := ci.Run(nil); code != 0 {
		output := done(t)
		t.Fatalf("bad: %d\n%s\n%s", code, output.Stdout(), output.Stderr())
	}

	view, done = testView(t)
	c := &UnlockCommand{
		Meta: Meta{
			View:       view,
			WorkingDir: td,
		},
	}

	// run with the incorrect lock ID
	args := []string{
		"-force",
		"LOCK_ID",
	}

	if code := c.Run(args); code == 0 {
		output := done(t)
		t.Fatalf("bad: %d\n%s\n%s", code, output.Stdout(), output.Stderr())
	}

	view, done = testView(t)
	c = &UnlockCommand{
		Meta: Meta{
			View:       view,
			WorkingDir: td,
		},
	}

	// lockID set in the test fixture
	args[1] = "2b6a6738-5dd5-50d6-c0ae-f6352977666b"
	if code := c.Run(args); code != 0 {
		output := done(t)
		t.Fatalf("bad: %d\n%s\n%s", code, output.Stdout(), output.Stderr())
	}
}
