package command

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
)

// Since we can't unlock a local state file, just test that calling unlock
// doesn't fail.
// TODO: mock remote state for UI testing
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
		err = terraform.WriteState(testState(), f)
		f.Close()
		if err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	p := testProvider()
	ui := new(cli.MockUi)
	c := &UnlockCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(p),
			Ui:          ui,
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
