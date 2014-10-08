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
	defer fixDir(tmp, cwd)

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

func TestPush_cliRemote_noState(t *testing.T) {
	tmp, cwd := testCwd(t)
	defer fixDir(tmp, cwd)

	s := terraform.NewState()
	conf, srv := testRemoteState(t, s, 200)
	defer srv.Close()

	ui := new(cli.MockUi)
	c := &PushCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(testProvider()),
			Ui:          ui,
		},
	}

	// Remote with no local state!
	args := []string{"-remote", conf.Name, "-remote-server", conf.Server}
	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}
}

func TestPush_local(t *testing.T) {
	tmp, cwd := testCwd(t)
	defer fixDir(tmp, cwd)

	s := terraform.NewState()
	s.Serial = 5
	conf, srv := testRemoteState(t, s, 200)

	s = terraform.NewState()
	s.Serial = 10
	s.Remote = conf
	defer srv.Close()

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
