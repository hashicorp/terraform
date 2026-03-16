// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"strings"
	"testing"

	"github.com/hashicorp/cli"
)

func TestInit_stateStoreBlockIsExperimental(t *testing.T) {

	t.Run("init command", func(t *testing.T) {
		// Create a temporary working directory with state_store in use
		td := t.TempDir()
		testCopyDir(t, testFixturePath("init-with-state-store"), td)
		t.Chdir(td)

		ui := new(cli.MockUi)
		view, done := testView(t)
		c := &InitCommand{
			Meta: Meta{
				Ui:                        ui,
				View:                      view,
				AllowExperimentalFeatures: false,
			},
		}

		args := []string{}
		code := c.Run(args)
		testOutput := done(t)
		if code != 1 {
			t.Fatalf("unexpected output: \n%s", testOutput.All())
		}

		// Check output
		output := testOutput.Stderr()
		if !strings.Contains(output, `Blocks of type "state_store" are not expected here`) {
			t.Fatalf("doesn't look like experiment is blocking access': %s", output)
		}
	})

	t.Run("non-init command: plan", func(t *testing.T) {
		// Create a temporary working directory with state_store in use
		td := t.TempDir()
		testCopyDir(t, testFixturePath("init-with-state-store"), td)
		t.Chdir(td)

		ui := new(cli.MockUi)
		view, done := testView(t)
		c := &PlanCommand{
			Meta: Meta{
				Ui:                        ui,
				View:                      view,
				AllowExperimentalFeatures: false,
			},
		}

		args := []string{}
		code := c.Run(args)
		testOutput := done(t)
		if code != 1 {
			t.Fatalf("unexpected output: \n%s", testOutput.All())
		}

		// Check output
		output := testOutput.Stderr()
		if !strings.Contains(output, `Blocks of type "state_store" are not expected here`) {
			t.Fatalf("doesn't look like experiment is blocking access': %s", output)
		}
	})

	t.Run("non-init command: state list", func(t *testing.T) {
		// Create a temporary working directory with state_store in use
		td := t.TempDir()
		testCopyDir(t, testFixturePath("init-with-state-store"), td)
		t.Chdir(td)

		ui := new(cli.MockUi)
		view, done := testView(t)
		c := &StateListCommand{
			Meta: Meta{
				Ui:                        ui,
				View:                      view,
				AllowExperimentalFeatures: false,
			},
		}

		args := []string{}
		code := c.Run(args)
		testOutput := done(t)
		if code != 1 {
			t.Fatalf("unexpected output: \n%s", testOutput.All())
		}

		// Check output
		output := ui.ErrorWriter.String()
		if !strings.Contains(output, `Blocks of type "state_store" are not expected here`) {
			t.Fatalf("doesn't look like experiment is blocking access': %s", output)
		}
	})
}
