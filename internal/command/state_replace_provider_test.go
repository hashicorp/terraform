package command

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mitchellh/cli"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/states"
)

func TestStateReplaceProvider(t *testing.T) {
	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "aws_instance",
				Name: "alpha",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"alpha","foo":"value","bar":"value"}`),
				Status:    states.ObjectReady,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("aws"),
				Module:   addrs.RootModule,
			},
		)
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "aws_instance",
				Name: "beta",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"beta","foo":"value","bar":"value"}`),
				Status:    states.ObjectReady,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("aws"),
				Module:   addrs.RootModule,
			},
		)
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "azurerm_virtual_machine",
				Name: "gamma",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"gamma","baz":"value"}`),
				Status:    states.ObjectReady,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewLegacyProvider("azurerm"),
				Module:   addrs.RootModule,
			},
		)
	})

	t.Run("happy path", func(t *testing.T) {
		statePath := testStateFile(t, state)

		ui := new(cli.MockUi)
		view, _ := testView(t)
		c := &StateReplaceProviderCommand{
			StateMeta{
				Meta: Meta{
					Ui:   ui,
					View: view,
				},
			},
		}

		inputBuf := &bytes.Buffer{}
		ui.InputReader = inputBuf
		inputBuf.WriteString("yes\n")

		args := []string{
			"-state", statePath,
			"hashicorp/aws",
			"acmecorp/aws",
		}
		if code := c.Run(args); code != 0 {
			t.Fatalf("return code: %d\n\n%s", code, ui.ErrorWriter.String())
		}

		testStateOutput(t, statePath, testStateReplaceProviderOutput)

		backups := testStateBackups(t, filepath.Dir(statePath))
		if len(backups) != 1 {
			t.Fatalf("unexpected backups: %#v", backups)
		}
		testStateOutput(t, backups[0], testStateReplaceProviderOutputOriginal)
	})

	t.Run("auto approve", func(t *testing.T) {
		statePath := testStateFile(t, state)

		ui := new(cli.MockUi)
		view, _ := testView(t)
		c := &StateReplaceProviderCommand{
			StateMeta{
				Meta: Meta{
					Ui:   ui,
					View: view,
				},
			},
		}

		inputBuf := &bytes.Buffer{}
		ui.InputReader = inputBuf

		args := []string{
			"-state", statePath,
			"-auto-approve",
			"hashicorp/aws",
			"acmecorp/aws",
		}
		if code := c.Run(args); code != 0 {
			t.Fatalf("return code: %d\n\n%s", code, ui.ErrorWriter.String())
		}

		testStateOutput(t, statePath, testStateReplaceProviderOutput)

		backups := testStateBackups(t, filepath.Dir(statePath))
		if len(backups) != 1 {
			t.Fatalf("unexpected backups: %#v", backups)
		}
		testStateOutput(t, backups[0], testStateReplaceProviderOutputOriginal)
	})

	t.Run("cancel at approval step", func(t *testing.T) {
		statePath := testStateFile(t, state)

		ui := new(cli.MockUi)
		view, _ := testView(t)
		c := &StateReplaceProviderCommand{
			StateMeta{
				Meta: Meta{
					Ui:   ui,
					View: view,
				},
			},
		}

		inputBuf := &bytes.Buffer{}
		ui.InputReader = inputBuf
		inputBuf.WriteString("no\n")

		args := []string{
			"-state", statePath,
			"hashicorp/aws",
			"acmecorp/aws",
		}
		if code := c.Run(args); code != 0 {
			t.Fatalf("return code: %d\n\n%s", code, ui.ErrorWriter.String())
		}

		testStateOutput(t, statePath, testStateReplaceProviderOutputOriginal)

		backups := testStateBackups(t, filepath.Dir(statePath))
		if len(backups) != 0 {
			t.Fatalf("unexpected backups: %#v", backups)
		}
	})

	t.Run("no matching provider found", func(t *testing.T) {
		statePath := testStateFile(t, state)

		ui := new(cli.MockUi)
		view, _ := testView(t)
		c := &StateReplaceProviderCommand{
			StateMeta{
				Meta: Meta{
					Ui:   ui,
					View: view,
				},
			},
		}

		args := []string{
			"-state", statePath,
			"hashicorp/google",
			"acmecorp/google",
		}
		if code := c.Run(args); code != 0 {
			t.Fatalf("return code: %d\n\n%s", code, ui.ErrorWriter.String())
		}

		testStateOutput(t, statePath, testStateReplaceProviderOutputOriginal)

		backups := testStateBackups(t, filepath.Dir(statePath))
		if len(backups) != 0 {
			t.Fatalf("unexpected backups: %#v", backups)
		}
	})

	t.Run("invalid flags", func(t *testing.T) {
		ui := new(cli.MockUi)
		view, _ := testView(t)
		c := &StateReplaceProviderCommand{
			StateMeta{
				Meta: Meta{
					Ui:   ui,
					View: view,
				},
			},
		}

		args := []string{
			"-invalid",
			"hashicorp/google",
			"acmecorp/google",
		}
		if code := c.Run(args); code == 0 {
			t.Fatalf("successful exit; want error")
		}

		if got, want := ui.ErrorWriter.String(), "Error parsing command-line flags"; !strings.Contains(got, want) {
			t.Fatalf("missing expected error message\nwant: %s\nfull output:\n%s", want, got)
		}
	})

	t.Run("wrong number of arguments", func(t *testing.T) {
		ui := new(cli.MockUi)
		view, _ := testView(t)
		c := &StateReplaceProviderCommand{
			StateMeta{
				Meta: Meta{
					Ui:   ui,
					View: view,
				},
			},
		}

		args := []string{"a", "b", "c", "d"}
		if code := c.Run(args); code == 0 {
			t.Fatalf("successful exit; want error")
		}

		if got, want := ui.ErrorWriter.String(), "Exactly two arguments expected"; !strings.Contains(got, want) {
			t.Fatalf("missing expected error message\nwant: %s\nfull output:\n%s", want, got)
		}
	})

	t.Run("invalid provider strings", func(t *testing.T) {
		ui := new(cli.MockUi)
		view, _ := testView(t)
		c := &StateReplaceProviderCommand{
			StateMeta{
				Meta: Meta{
					Ui:   ui,
					View: view,
				},
			},
		}

		args := []string{
			"hashicorp/google_cloud",
			"-/-/google",
		}
		if code := c.Run(args); code == 0 {
			t.Fatalf("successful exit; want error")
		}

		got := ui.ErrorWriter.String()
		msgs := []string{
			`Invalid "from" provider "hashicorp/google_cloud"`,
			"Invalid provider type",
			`Invalid "to" provider "-/-/google"`,
			"Invalid provider source hostname",
		}
		for _, msg := range msgs {
			if !strings.Contains(got, msg) {
				t.Errorf("missing expected error message\nwant: %s\nfull output:\n%s", msg, got)
			}
		}
	})
}

func TestStateReplaceProvider_docs(t *testing.T) {
	c := &StateReplaceProviderCommand{}

	if got, want := c.Help(), "Usage: terraform [global options] state replace-provider"; !strings.Contains(got, want) {
		t.Fatalf("unexpected help text\nwant: %s\nfull output:\n%s", want, got)
	}

	if got, want := c.Synopsis(), "Replace provider in the state"; got != want {
		t.Fatalf("unexpected synopsis\nwant: %s\nfull output:\n%s", want, got)
	}
}

const testStateReplaceProviderOutputOriginal = `
aws_instance.alpha:
  ID = alpha
  provider = provider["registry.terraform.io/hashicorp/aws"]
  bar = value
  foo = value
aws_instance.beta:
  ID = beta
  provider = provider["registry.terraform.io/hashicorp/aws"]
  bar = value
  foo = value
azurerm_virtual_machine.gamma:
  ID = gamma
  provider = provider["registry.terraform.io/-/azurerm"]
  baz = value
`

const testStateReplaceProviderOutput = `
aws_instance.alpha:
  ID = alpha
  provider = provider["registry.terraform.io/acmecorp/aws"]
  bar = value
  foo = value
aws_instance.beta:
  ID = beta
  provider = provider["registry.terraform.io/acmecorp/aws"]
  bar = value
  foo = value
azurerm_virtual_machine.gamma:
  ID = gamma
  provider = provider["registry.terraform.io/-/azurerm"]
  baz = value
`
