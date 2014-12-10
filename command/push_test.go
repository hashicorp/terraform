package command

import (
	"bytes"
	"testing"

	"github.com/hashicorp/terraform/remote"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
)

func TestPush_noRemote(t *testing.T) {
	tmp, cwd := testCwd(t)
	defer testFixCwd(t, tmp, cwd)

	ui := new(cli.MockUi)
	c := &PushCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(testProvider()),
			Ui:          ui,
		},
	}

	args := []string{}
	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}
}

func TestPush_local(t *testing.T) {
	tmp, cwd := testCwd(t)
	defer testFixCwd(t, tmp, cwd)

	s := terraform.NewState()
	s.Serial = 5
	conf, srv := testRemoteState(t, s, 200)
	defer srv.Close()

	s = terraform.NewState()
	s.Serial = 10
	s.Remote = conf

	// Store the local state
	buf := bytes.NewBuffer(nil)
	terraform.WriteState(s, buf)
	remote.EnsureDirectory()
	remote.Persist(buf)

	ui := new(cli.MockUi)
	c := &PushCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(testProvider()),
			Ui:          ui,
		},
	}
	args := []string{}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}
}
