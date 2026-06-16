// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/cli"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/workdir"
	"github.com/hashicorp/terraform/internal/providers"
)

func TestStateMigrate_fromBackendToBackend(t *testing.T) {
	wd := tempWorkingDirFixture(t, "state-migrate-backend-to-backend")
	t.Chdir(wd.RootModuleDir())

	ui := cli.NewMockUi()
	view, done := testView(t)
	c := &StateMigrateCommand{
		Meta: Meta{
			Ui:                        ui,
			View:                      view,
			WorkingDir:                wd,
			AllowExperimentalFeatures: true,
		},
	}

	args := []string{"-no-color"}
	code := c.Run(args)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	out := done(t)

	expectedMsg := `Migrating state from backend "local" to backend "local"...`
	if !strings.Contains(out.Stdout(), expectedMsg) {
		t.Fatalf("expected output %q, got %q", expectedMsg, out.Stdout())
	}

	expectedErr := "Not implemented yet"
	if !strings.Contains(out.Stderr(), expectedErr) {
		t.Fatalf("expected output %q, got %q", expectedErr, out.Stderr())
	}
}

func TestStateMigrate_fromBackendToStateStore(t *testing.T) {
	wd := tempWorkingDirFixture(t, "state-migrate-backend-to-state-store")
	t.Chdir(wd.RootModuleDir())

	mockProvider := mockPluggableStateStorageProvider()
	mockProviderAddress := addrs.NewDefaultProvider("test")
	providerSource := newMockProviderSource(t, map[string][]string{
		"hashicorp/test": {"1.2.3"}, // Version referenced in text fixtures
	})

	ui := cli.NewMockUi()
	view, done := testView(t)
	c := &StateMigrateCommand{
		Meta: Meta{
			Ui:                        ui,
			View:                      view,
			WorkingDir:                wd,
			AllowExperimentalFeatures: true,
			ProviderSource:            providerSource,
			testingOverrides: &testingOverrides{
				Providers: map[addrs.Provider]providers.Factory{
					mockProviderAddress: providers.FactoryFixed(mockProvider),
				},
			},
		},
	}

	args := []string{"-no-color"}
	code := c.Run(args)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	out := done(t)

	expectedMsgs := []string{
		`Migrating state from backend "local" to state store "test_store" (registry.terraform.io/hashicorp/test)...`,
		`Initializing provider plugin for state migration destination...- Reusing previous version of hashicorp/test from the dependency lock file- Installing hashicorp/test v1.2.3...- Installed hashicorp/test v1.2.3 (verified checksum)`,
	}
	for _, expectedMsg := range expectedMsgs {
		if !strings.Contains(out.Stdout(), expectedMsg) {
			t.Fatalf("expected output %q, got %q", expectedMsg, out.Stdout())
		}
	}

	unexpectedMsgs := []string{
		// This scenario involves migrating from a backend, not from a state store.
		`Initializing provider plugin for state migration source...`,
	}
	for _, unexpectedMsg := range unexpectedMsgs {
		if strings.Contains(out.Stdout(), unexpectedMsg) {
			t.Fatalf("unexpected output %q, got %q", unexpectedMsg, out.Stdout())
		}
	}

	expectedErr := "Not implemented yet"
	if !strings.Contains(out.Stderr(), expectedErr) {
		t.Fatalf("expected output %q, got %q", expectedErr, out.All())
	}
}

func TestStateMigrate_fromStateStoreToStateStore(t *testing.T) {
	wd := tempWorkingDirFixture(t, "state-migrate-state-store-to-state-store")
	t.Chdir(wd.RootModuleDir())

	mockProvider := mockPluggableStateStorageProvider()
	mockProviderAddress := addrs.NewDefaultProvider("test")
	providerSource := newMockProviderSource(t, map[string][]string{
		"hashicorp/test": {"1.2.3"}, // Version referenced in text fixtures
	})

	ui := cli.NewMockUi()
	view, done := testView(t)
	c := &StateMigrateCommand{
		Meta: Meta{
			Ui:                        ui,
			View:                      view,
			WorkingDir:                wd,
			AllowExperimentalFeatures: true,
			ProviderSource:            providerSource,
			testingOverrides: &testingOverrides{
				Providers: map[addrs.Provider]providers.Factory{
					mockProviderAddress: providers.FactoryFixed(mockProvider),
				},
			},
		},
	}

	args := []string{"-no-color"}
	code := c.Run(args)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	out := done(t)

	expectedMsgs := []string{
		`Migrating state from state store "test_store" (registry.terraform.io/hashicorp/test) to state store "test_store" (registry.terraform.io/hashicorp/test)...`,
		`Initializing provider plugin for state migration source...- Reusing previous version of hashicorp/test from the dependency lock file- Installing hashicorp/test v1.2.3...- Installed hashicorp/test v1.2.3 (verified checksum)`,
		`Initializing provider plugin for state migration destination...- Reusing previous version of hashicorp/test from the dependency lock file- Installing hashicorp/test v1.2.3...- Installed hashicorp/test v1.2.3 (verified checksum)`,
	}
	for _, expectedMsg := range expectedMsgs {
		if !strings.Contains(out.Stdout(), expectedMsg) {
			t.Fatalf("expected output %q, got %q", expectedMsg, out.Stdout())
		}
	}

	expectedErr := "Not implemented yet"
	if !strings.Contains(out.Stderr(), expectedErr) {
		t.Fatalf("expected output %q, got %q", expectedErr, out.All())
	}
}

func TestStateMigrate_fromStateStoreToBackend(t *testing.T) {
	wd := tempWorkingDirFixture(t, "state-migrate-state-store-to-backend")
	t.Chdir(wd.RootModuleDir())

	ui := cli.NewMockUi()
	view, done := testView(t)
	c := &StateMigrateCommand{
		Meta: Meta{
			Ui:                        ui,
			View:                      view,
			WorkingDir:                wd,
			AllowExperimentalFeatures: true,
		},
	}

	args := []string{"-no-color"}
	code := c.Run(args)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	out := done(t)

	expectedMsg := `Migrating state from state store "test_store" (registry.terraform.io/hashicorp/test) to backend "local"...`
	if !strings.Contains(out.Stdout(), expectedMsg) {
		t.Fatalf("expected output %q, got %q", expectedMsg, out.Stdout())
	}

	expectedErr := "Not implemented yet"
	if !strings.Contains(out.Stderr(), expectedErr) {
		t.Fatalf("expected output %q, got %q", expectedErr, out.All())
	}
}

func TestStateMigrate_missingModuleFiles(t *testing.T) {
	wd := tempWorkingDirFixture(t, "state-migrate-missing-mod-files")
	t.Chdir(wd.RootModuleDir())

	ui := cli.NewMockUi()
	view, done := testView(t)
	c := &StateMigrateCommand{
		Meta: Meta{
			Ui:                        ui,
			View:                      view,
			WorkingDir:                wd,
			AllowExperimentalFeatures: true,
		},
	}

	args := []string{
		"-input=false",
		"-no-color",
	}
	code := c.Run(args)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	out := done(t)

	expectedMsg := "Error: Unknown migration destination"
	if !strings.Contains(out.Stderr(), expectedMsg) {
		t.Fatalf("expected output %q, got %q", expectedMsg, out.All())
	}
}

func TestStateMigrate_emptyModuleFiles(t *testing.T) {
	wd := tempWorkingDirFixture(t, "state-migrate-empty-mod-files")
	t.Chdir(wd.RootModuleDir())

	ui := cli.NewMockUi()
	view, done := testView(t)
	c := &StateMigrateCommand{
		Meta: Meta{
			Ui:                        ui,
			View:                      view,
			WorkingDir:                wd,
			AllowExperimentalFeatures: true,
		},
	}

	args := []string{
		"-input=false",
		"-no-color",
	}
	code := c.Run(args)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	out := done(t)

	expectedMsg := "Error: Unknown migration destination"
	if !strings.Contains(out.Stderr(), expectedMsg) {
		t.Fatalf("expected output %q, got %q", expectedMsg, out.All())
	}
}

func TestStateMigrate_missingMigrationFiles(t *testing.T) {
	wd := tempWorkingDirFixture(t, "state-migrate-missing-migrate-files")
	t.Chdir(wd.RootModuleDir())

	ui := cli.NewMockUi()
	view, done := testView(t)
	c := &StateMigrateCommand{
		Meta: Meta{
			Ui:                        ui,
			View:                      view,
			WorkingDir:                wd,
			AllowExperimentalFeatures: true,
		},
	}

	args := []string{
		"-input=false",
		"-no-color",
	}
	code := c.Run(args)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	out := done(t)

	expectedMsg := "Error: No state migration instructions found"
	if !strings.Contains(out.Stderr(), expectedMsg) {
		t.Fatalf("expected output %q, got %q", expectedMsg, out.All())
	}
}

func TestStateMigrate_nonExistentLockFiles(t *testing.T) {
	// use temporary directory to ensure the lock files certainly do not exist
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	ui := cli.NewMockUi()
	view, done := testView(t)
	c := &StateMigrateCommand{
		Meta: Meta{
			Ui:                        ui,
			View:                      view,
			WorkingDir:                workdir.NewDir(tmpDir),
			AllowExperimentalFeatures: true,
		},
	}

	args := []string{
		"-input=false",
		"-source-provider-lock-file", filepath.Join(tmpDir, ".terraform.lock.hcl"),
		"-destination-provider-lock-file", filepath.Join(tmpDir, ".terraform.lock.hcl"),
	}
	code := c.Run(args)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	out := done(t)

	expectedMsg := "Unreadable source provider lock file"
	if !strings.Contains(out.Stderr(), expectedMsg) {
		t.Fatalf("expected output %q, got %q", expectedMsg, out.All())
	}

	expectedMsg = "Unreadable destination provider lock file"
	if !strings.Contains(out.Stderr(), expectedMsg) {
		t.Fatalf("expected output %q, got %q", expectedMsg, out.All())
	}
}
