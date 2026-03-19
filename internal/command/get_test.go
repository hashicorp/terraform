// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"strings"
	"testing"

	"github.com/hashicorp/cli"

	"github.com/hashicorp/terraform/internal/backend"
	backendInit "github.com/hashicorp/terraform/internal/backend/init"
	backendCloud "github.com/hashicorp/terraform/internal/cloud"
)

func TestGet(t *testing.T) {
	wd := tempWorkingDirFixture(t, "get")
	t.Chdir(wd.RootModuleDir())

	ui := cli.NewMockUi()
	c := &GetCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			WorkingDir:       wd,
		},
	}

	args := []string{}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	output := ui.OutputWriter.String()
	if !strings.Contains(output, "- foo in") {
		t.Fatalf("doesn't look like get: %s", output)
	}
}

func TestGet_multipleArgs(t *testing.T) {
	wd := tempWorkingDir(t)
	t.Chdir(wd.RootModuleDir())

	ui := cli.NewMockUi()
	c := &GetCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			WorkingDir:       wd,
		},
	}

	args := []string{
		"bad",
		"bad",
	}
	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: \n%s", ui.OutputWriter.String())
	}
}

func TestGet_update(t *testing.T) {
	wd := tempWorkingDirFixture(t, "get")
	t.Chdir(wd.RootModuleDir())

	ui := cli.NewMockUi()
	c := &GetCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			WorkingDir:       wd,
		},
	}

	args := []string{
		"-update",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	output := ui.OutputWriter.String()
	if !strings.Contains(output, `- foo in`) {
		t.Fatalf("doesn't look like get: %s", output)
	}
}

func TestGet_cancel(t *testing.T) {
	// This test runs `terraform get` as if SIGINT (or similar on other
	// platforms) were sent to it, testing that it is interruptible.

	wd := tempWorkingDirFixture(t, "init-registry-module")
	t.Chdir(wd.RootModuleDir())

	// Our shutdown channel is pre-closed so init will exit as soon as it
	// starts a cancelable portion of the process.
	shutdownCh := make(chan struct{})
	close(shutdownCh)

	ui := cli.NewMockUi()
	c := &GetCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			WorkingDir:       wd,
			ShutdownCh:       shutdownCh,
		},
	}

	args := []string{}
	if code := c.Run(args); code == 0 {
		t.Fatalf("succeeded; wanted error\n%s", ui.OutputWriter.String())
	}

	if got, want := ui.ErrorWriter.String(), `Module installation was canceled by an interrupt signal`; !strings.Contains(got, want) {
		t.Fatalf("wrong error message\nshould contain: %s\ngot:\n%s", want, got)
	}
}

func TestGet_constVariable(t *testing.T) {
	// Scenario 1: no value for variable -> diagnostic
	t.Run("missing value", func(t *testing.T) {
		wd := tempWorkingDirFixture(t, "dynamic-module-sources/get-const-var")
		t.Chdir(wd.RootModuleDir())

		ui := cli.NewMockUi()
		c := &GetCommand{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(testProvider()),
				Ui:               ui,
				WorkingDir:       wd,
			},
		}

		args := []string{}
		if code := c.Run(args); code == 0 {
			t.Fatalf("expected error, got 0")
		}

		errStr := ui.ErrorWriter.String()
		if !strings.Contains(errStr, "No value for required variable") {
			t.Fatalf("expected missing variable error, got: %s", errStr)
		}
	})

	// Scenario 2: value via cli -> works
	t.Run("value via cli", func(t *testing.T) {
		wd := tempWorkingDirFixture(t, "dynamic-module-sources/get-const-var")
		t.Chdir(wd.RootModuleDir())

		ui := cli.NewMockUi()
		c := &GetCommand{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(testProvider()),
				Ui:               ui,
				WorkingDir:       wd,
			},
		}

		args := []string{"-var", "module_name=example"}
		if code := c.Run(args); code != 0 {
			t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
		}

		output := ui.OutputWriter.String()
		if !strings.Contains(output, "- example in") {
			t.Fatalf("doesn't look like get: %s", output)
		}
	})

	// Scenario 3: value via backend
	t.Run("value via backend", func(t *testing.T) {
		server := cloudTestServerWithVars(t)
		defer server.Close()
		d := testDisco(server)

		previousBackend := backendInit.Backend("cloud")
		backendInit.Set("cloud", func() backend.Backend { return backendCloud.New(d) })
		defer backendInit.Set("cloud", previousBackend)

		wd := tempWorkingDirFixture(t, "dynamic-module-sources/get-const-var-backend")
		t.Chdir(wd.RootModuleDir())

		ui := cli.NewMockUi()
		c := &GetCommand{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(testProvider()),
				Ui:               ui,
				WorkingDir:       wd,
				Services:         d,
			},
		}

		args := []string{}
		if code := c.Run(args); code != 0 {
			t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
		}

		output := ui.OutputWriter.String()
		if !strings.Contains(output, "- example in") {
			t.Fatalf("doesn't look like get: %s", output)
		}
	})
}
