// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"strings"
	"testing"

	"github.com/hashicorp/cli"
)

func TestInit_stateStoreBlockIsExperimental(t *testing.T) {

	// When experiments are enabled, users are prompted to add experiments = [pluggable_state_stores] to their config.
	t.Run("init command without `pluggable_state_stores` experiment in config", func(t *testing.T) {
		// Create a temporary working directory with state_store in use but experiment not declared
		td := t.TempDir()
		testCopyDir(t, testFixturePath("init-with-state-store-no-experiment"), td)
		t.Chdir(td)

		ui := new(cli.MockUi)
		view, done := testView(t)
		c := &InitCommand{
			Meta: Meta{
				Ui:                        ui,
				View:                      view,
				AllowExperimentalFeatures: true,
			},
		}

		args := []string{}
		code := c.Run(args)
		testOutput := done(t)
		if code != 1 {
			t.Fatalf("unexpected output: \n%s", testOutput.All())
		}

		// Check output
		output := cleanString(testOutput.Stderr())
		if !strings.Contains(output, `Error: Pluggable state store experiment not supported`) {
			t.Fatalf("doesn't look like experiment is blocking access': %s", output)
		}
		if !strings.Contains(output, "opt into the \"pluggable_state_stores\" experiment using the `terraform` block's `experiments` attribute") {
			t.Fatalf("expected the error to explain the need for a config change': %s", output)
		}
	})

	// When experiments aren't enabled, the state_store block is reported as being unexpected
	t.Run("init command without experiments enabled", func(t *testing.T) {
		// Create a temporary working directory with state_store in use but experiment not declared
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

	t.Run("non-init command: `plan` without experiments enabled", func(t *testing.T) {
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

	t.Run("non-init command: `state list` without experiments enabled", func(t *testing.T) {
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
