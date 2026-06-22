// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/cli"
	"github.com/hashicorp/terraform/internal/command/workdir"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/hashicorp/terraform/internal/states/statefile"
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

	_ = testInputMap(t, map[string]string{
		"backend-migrate-copy-to-empty": "yes",
	})

	args := []string{"-no-color"}
	code := c.Run(args)
	out := done(t)
	if code != 0 {
		t.Fatalf("expected exit code 1, got %d\nstderr: %q", code, out.Stderr())
	}

	expectedMsg := `Finished migrating state from backend "local" to backend "local"...`
	if !strings.Contains(out.Stdout(), expectedMsg) {
		t.Fatalf("expected output %q, got %q", expectedMsg, out.Stdout())
	}

	f, err := os.Open("destination-backend.tfstate")
	if err != nil {
		t.Fatalf("failed to read migrated state: %s", err)
	}
	t.Cleanup(func() {
		err := f.Close()
		if err != nil {
			t.Fatal(err)
		}
	})
	s, err := statefile.Read(f)
	if err != nil {
		t.Fatal(err)
	}
	_, ok := s.State.RootOutputValues["test"]
	if !ok {
		t.Fatalf("unable to find test output in migrated state")
	}
}

func TestStateMigrate_fromBackendToStateStore(t *testing.T) {
	wd := tempWorkingDirFixture(t, "state-migrate-backend-to-state-store")
	t.Chdir(wd.RootModuleDir())

	p := mockPluggableStateStorageProvider(mockSingleStateStoreSchema("test_store"))
	p.MockStates = testing_provider.NewMockStateBytesWithStateIds("test_store", []string{"default"})
	providerSource := newMockProviderSource(t, map[string][]string{
		"hashicorp/test": {"1.0.0"},
	})

	ui := cli.NewMockUi()
	view, done := testView(t)
	c := &StateMigrateCommand{
		Meta: Meta{
			Ui:                        ui,
			View:                      view,
			WorkingDir:                wd,
			AllowExperimentalFeatures: true,
			testingOverrides:          metaOverridesForProvider(p),
			ProviderSource:            providerSource,
		},
	}

	_ = testInputMap(t, map[string]string{
		"backend-migrate-copy-to-empty": "yes",
	})

	args := []string{"-no-color"}
	code := c.Run(args)
	out := done(t)
	if code != 0 {
		t.Fatalf("unexpected exit: %d\nstderr: %q", code, out.Stderr())
	}

	expectedMsg := `Migrating state from backend "local" to state store "test_store" (hashicorp/test)...`
	if !strings.Contains(out.Stdout(), expectedMsg) {
		t.Fatalf("expected output %q, got %q", expectedMsg, out.Stdout())
	}

	b, err := p.MockStates.Read("test_store", "default")
	if err != nil {
		t.Fatalf("unable to find migrated state in mock provider: %s", err)
	}
	s, err := statefile.Read(bytes.NewBuffer(b))
	if err != nil {
		t.Fatal(err)
	}
	_, ok := s.State.RootOutputValues["test"]
	if !ok {
		t.Fatalf("unable to find test output in migrated state")
	}
}

func TestStateMigrate_fromStateStoreToStateStore(t *testing.T) {
	wd := tempWorkingDirFixture(t, "state-migrate-state-store-to-state-store")
	t.Chdir(wd.RootModuleDir())

	b, err := os.ReadFile("source-pss.tfstate")
	if err != nil {
		t.Fatal(err)
	}

	pssSchemas := map[string]providers.Schema{
		"test_src": {
			Body: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{},
			},
		},
		"test_dst": {
			Body: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{},
			},
		},
	}
	p := mockPluggableStateStorageProvider(pssSchemas)
	p.MockStates = testing_provider.MockStateBytes{
		"test_src": map[string][]byte{"default": []byte(b)},
		"test_dst": map[string][]byte{},
	}
	providerSource := newMockProviderSource(t, map[string][]string{
		"hashicorp/test": {"1.0.0"},
	})

	ui := cli.NewMockUi()
	view, done := testView(t)
	c := &StateMigrateCommand{
		Meta: Meta{
			Ui:                        ui,
			View:                      view,
			WorkingDir:                wd,
			AllowExperimentalFeatures: true,
			testingOverrides:          metaOverridesForProvider(p),
			ProviderSource:            providerSource,
		},
	}

	_ = testInputMap(t, map[string]string{
		"backend-migrate-copy-to-empty": "yes",
	})

	args := []string{"-no-color"}
	code := c.Run(args)
	out := done(t)
	if code != 0 {
		t.Fatalf("unexpected exit: %d\nstderr: %q", code, out.Stderr())
	}

	expectedMsg := `Migrating state from state store "test_src" (hashicorp/test) to state store "test_dst" (hashicorp/test)...`
	if !strings.Contains(out.Stdout(), expectedMsg) {
		t.Fatalf("expected output %q, got %q", expectedMsg, out.Stdout())
	}

	b, err = p.MockStates.Read("test_dst", "default")
	if err != nil {
		t.Fatalf("unable to find migrated state in mock provider: %s", err)
	}
	s, err := statefile.Read(bytes.NewBuffer(b))
	if err != nil {
		t.Fatal(err)
	}
	_, ok := s.State.RootOutputValues["test"]
	if !ok {
		t.Fatalf("unable to find test output in migrated state")
	}
}

func TestStateMigrate_fromStateStoreToBackend(t *testing.T) {
	wd := tempWorkingDirFixture(t, "state-migrate-state-store-to-backend")
	t.Chdir(wd.RootModuleDir())

	p := mockPluggableStateStorageProvider(mockSingleStateStoreSchema("test_store"))
	b, err := os.ReadFile("source-pss.tfstate")
	if err != nil {
		t.Fatal(err)
	}
	p.MockStates = testing_provider.NewMockStateBytesWithSingleState(
		"test_store",
		"default",
		b,
	)

	providerSource := newMockProviderSource(t, map[string][]string{
		"hashicorp/test": {"1.0.0"},
	})

	ui := cli.NewMockUi()
	view, done := testView(t)
	c := &StateMigrateCommand{
		Meta: Meta{
			Ui:                        ui,
			View:                      view,
			WorkingDir:                wd,
			AllowExperimentalFeatures: true,
			testingOverrides:          metaOverridesForProvider(p),
			ProviderSource:            providerSource,
		},
	}

	_ = testInputMap(t, map[string]string{
		"backend-migrate-copy-to-empty": "yes",
	})

	args := []string{"-no-color"}
	code := c.Run(args)
	out := done(t)
	if code != 0 {
		t.Fatalf("unexpected exit: %d\nstderr: %q", code, out.Stderr())
	}

	expectedMsg := `Migrating state from state store "test_store" (hashicorp/test) to backend "local"...`
	if !strings.Contains(out.Stdout(), expectedMsg) {
		t.Fatalf("expected output %q, got %q", expectedMsg, out.Stdout())
	}

	f, err := os.Open("destination-backend.tfstate")
	if err != nil {
		t.Fatalf("failed to read migrated state: %s", err)
	}
	t.Cleanup(func() {
		err := f.Close()
		if err != nil {
			t.Fatal(err)
		}
	})
	s, err := statefile.Read(f)
	if err != nil {
		t.Fatal(err)
	}
	_, ok := s.State.RootOutputValues["test"]
	if !ok {
		t.Fatalf("unable to find test output in migrated state")
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
