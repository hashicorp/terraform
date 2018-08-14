package command

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform/backend/remote-state/inmem"
	"github.com/hashicorp/terraform/helper/copy"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
)

// Since we can't unlock a local state file, just test that calling unlock
// doesn't fail.
func TestUnlock(t *testing.T) {
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
		err = terraform.WriteState(terraform.NewState(), f)
		f.Close()
		if err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	p := testProvider()
	ui := new(cli.MockUi)
	c := &UnlockCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	args := []string{
		"-force",
		"LOCK_ID",
	}

	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: %d\n%s\n%s", code, ui.OutputWriter.String(), ui.ErrorWriter.String())
	}
}

// Newly configured backend
func TestUnlock_inmemBackend(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("backend-inmem-locked"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()
	defer inmem.Reset()

	// init backend
	ui := new(cli.MockUi)
	ci := &InitCommand{
		Meta: Meta{
			Ui: ui,
		},
	}
	if code := ci.Run(nil); code != 0 {
		t.Fatalf("bad: %d\n%s", code, ui.ErrorWriter)
	}

	ui = new(cli.MockUi)
	c := &UnlockCommand{
		Meta: Meta{
			Ui: ui,
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
			Ui: ui,
		},
	}

	// lockID set in the test fixture
	args[1] = "2b6a6738-5dd5-50d6-c0ae-f6352977666b"
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n%s\n%s", code, ui.OutputWriter.String(), ui.ErrorWriter.String())
	}

}
