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

// More thorough tests for providers mirror can be found in the e2etest
func TestProvidersMirror(t *testing.T) {
	// noop example
	t.Run("noop", func(t *testing.T) {
		c := &ProvidersMirrorCommand{}
		code := c.Run([]string{"."})
		if code != 0 {
			t.Fatalf("wrong exit code. expected 0, got %d", code)
		}
	})

	t.Run("missing arg error", func(t *testing.T) {
		ui := new(cli.MockUi)
		c := &ProvidersMirrorCommand{
			Meta: Meta{Ui: ui},
		}
		code := c.Run([]string{})
		if code != 1 {
			t.Fatalf("wrong exit code. expected 1, got %d", code)
		}

		got := ui.ErrorWriter.String()
		if !strings.Contains(got, "Error: No output directory specified") {
			t.Fatalf("missing directory error from output, got:\n%s\n", got)
		}
	})
}

func TestProvidersMirror_constVariable(t *testing.T) {
	t.Run("missing value", func(t *testing.T) {
		wd := tempWorkingDirFixture(t, "dynamic-module-sources/command-with-const-var")
		t.Chdir(wd.RootModuleDir())

		ui := cli.NewMockUi()
		c := &ProvidersMirrorCommand{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(testProvider()),
				Ui:               ui,
				WorkingDir:       wd,
			},
		}

		args := []string{t.TempDir()}
		if code := c.Run(args); code == 0 {
			t.Fatalf("expected error, got 0")
		}

		errStr := ui.ErrorWriter.String()
		if !strings.Contains(errStr, "No value for required variable") {
			t.Fatalf("expected missing variable error, got: %s", errStr)
		}
	})

	t.Run("value via cli", func(t *testing.T) {
		// We'll reuse our cloud test server, so Terraform has at least some services available
		server := cloudTestServerWithVars(t)
		defer server.Close()
		d := testDisco(server)

		wd := tempWorkingDirFixture(t, "dynamic-module-sources/command-with-const-var")
		t.Chdir(wd.RootModuleDir())

		ui := cli.NewMockUi()
		c := &ProvidersMirrorCommand{
			Meta: Meta{
				Ui:         ui,
				WorkingDir: wd,
				Services:   d,
			},
		}

		args := []string{"-var", "module_name=child", t.TempDir()}
		if code := c.Run(args); code == 0 {
			t.Fatalf("expected error, got 0")
		}

		// We expect an error, since the test provider can't be found on the registry
		errStr := ui.ErrorWriter.String()
		if !strings.Contains(errStr, "Error: Provider not available") {
			t.Fatalf("expected provider not found error, got: %s", errStr)
		}
	})

	t.Run("value via backend", func(t *testing.T) {
		server := cloudTestServerWithVars(t)
		defer server.Close()
		d := testDisco(server)

		previousBackend := backendInit.Backend("cloud")
		backendInit.Set("cloud", func() backend.Backend { return backendCloud.New(d) })
		defer backendInit.Set("cloud", previousBackend)

		wd := tempWorkingDirFixture(t, "dynamic-module-sources/command-with-const-var-cloud-backend")
		t.Chdir(wd.RootModuleDir())

		ui := cli.NewMockUi()
		c := &ProvidersMirrorCommand{
			Meta: Meta{
				Ui:         ui,
				WorkingDir: wd,
				Services:   d,
			},
		}

		args := []string{t.TempDir()}
		if code := c.Run(args); code == 0 {
			t.Fatalf("expected error, got 0")
		}

		// We expect an error, since the test provider can't be found on the registry
		errStr := ui.ErrorWriter.String()
		if !strings.Contains(errStr, "Error: Provider not available") {
			t.Fatalf("expected provider not found error, got: %s", errStr)
		}
	})
}
