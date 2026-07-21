// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/cli"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/clistate"
	"github.com/hashicorp/terraform/internal/command/workdir"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/getproviders"
	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/hashicorp/terraform/internal/states/statefile"
)

func TestStateMigrate_fromBackendToBackend(t *testing.T) {
	fixture := "state-migrate-backend-to-backend"
	wd := tempWorkingDirFixture(t, fixture)
	t.Chdir(wd.RootModuleDir())

	ui := testUiWrapped(t)
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

	// Assert expected human output is made
	checkGoldenReferenceHumanOutput(t, out, fixture)

	// Assert the migrated state contains expected content
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

	// Assert the backend state file describes the new backend location,
	statePath := filepath.Join(c.DataDir(), DefaultStateFilename)
	sMgr := &clistate.LocalState{Path: statePath}
	if err := sMgr.RefreshState(); err != nil {
		t.Fatal(err)
	}
	backendState := sMgr.State()
	if backendState.StateStore != nil {
		t.Fatalf("expected backend state file to not describe a state_store during backend=>backend, but it's set")
	}
	if backendState.Backend == nil {
		t.Fatalf("expected backend state file to describe a backend during backend=>backend, but it's nil")
	}
	if backendState.Backend.Type != "local" {
		t.Fatalf("expected backend state file to describe the destination backend \"local\", but got %q", backendState.Backend.Type)
	}
	if got, want := normalizeJSON(t, backendState.Backend.ConfigRaw), `{"path":"destination-backend.tfstate","workspace_dir":null}`; got != want {
		t.Errorf("wrong config\ngot:  %s\nwant: %s", got, want)
	}
}

func TestStateMigrate_fromBackendToStateStore(t *testing.T) {
	fixture := "state-migrate-backend-to-state-store"
	wd := tempWorkingDirFixture(t, fixture)
	t.Chdir(wd.RootModuleDir())

	p := mockPluggableStateStorageProvider(mockSingleStateStoreSchema("test_store"))
	p.MockStates = testing_provider.NewMockStateBytesWithStateIds("test_store", []string{"default"})
	providerSource := newMockProviderSource(t, map[string][]string{
		"hashicorp/test": {"1.2.3"},
	})

	ui := testUiWrapped(t)
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

	// Assert expected human output is made
	checkGoldenReferenceHumanOutput(t, out, fixture)

	// Assert the migrated state contains expected content
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

	// Assert the backend state file describes the new state store location.
	statePath := filepath.Join(c.DataDir(), DefaultStateFilename)
	sMgr := &clistate.LocalState{Path: statePath}
	if err := sMgr.RefreshState(); err != nil {
		t.Fatal(err)
	}
	backendState := sMgr.State()
	if backendState.Backend != nil {
		t.Fatalf("expected backend state file to not describe a backend during backend=>state_store migration, but it's set")
	}
	if backendState.StateStore == nil {
		t.Fatalf("expected backend state file to describe a state store during backend=>state_store migration, but it's nil")
	}
	if backendState.StateStore.Provider.Source.Type != "test" {
		t.Fatalf("expected backend state file to describe the destination state store provider \"test\", but got %q", backendState.StateStore.Provider)
	}
	if backendState.StateStore.Provider.Version.String() != "1.2.3" {
		t.Fatalf("expected backend state file to describe the destination state store provider version \"1.2.3\", but got %q", backendState.StateStore.Provider.Version)
	}
	if backendState.StateStore.Type != "test_store" {
		t.Fatalf("expected backend state file to describe the destination state store type \"test_store\", but got %q", backendState.StateStore.Type)
	}
}

// Testing migration between two state stores in a single provider.
// Different cases describe whether the source provider is already in the dependency lock file or not.
func TestStateMigrate_fromStateStoreToStateStore_inSingleProvider(t *testing.T) {
	// All test cases perform a migration from test_src to test_dst store implementations in v1.2.3 of hashicorp/test provider.
	// So a common assertion func can be re-used.
	assertBackendStateFile := func(t *testing.T, c *StateMigrateCommand) {
		statePath := filepath.Join(c.DataDir(), DefaultStateFilename)
		sMgr := &clistate.LocalState{Path: statePath}
		if err := sMgr.RefreshState(); err != nil {
			t.Fatal(err)
		}

		backendState := sMgr.State()
		if backendState.Backend != nil {
			t.Fatalf("expected backend state file to not describe a backend during state_store=>state_store migration, but it's set")
		}
		if backendState.StateStore == nil {
			t.Fatalf("expected backend state file to describe a state store during state_store=>state_store migration, but it's nil")
		}
		if backendState.StateStore.Provider.Source.Type != "test" {
			t.Fatalf("expected backend state file to describe the destination state store provider \"test\", but got %q", backendState.StateStore.Provider)
		}
		if backendState.StateStore.Provider.Version.String() != "1.2.3" {
			t.Fatalf("expected backend state file to describe the destination state store provider version \"1.2.3\", but got %q", backendState.StateStore.Provider.Version)
		}
		if backendState.StateStore.Type != "test_dst" {
			t.Fatalf("expected backend state file to describe the destination state store type \"test_dst\", but got %q", backendState.StateStore.Type)
		}
		// The state store schema is empty, so not even any null fields in expected JSON
		if got, want := normalizeJSON(t, backendState.StateStore.ConfigRaw), `{}`; got != want {
			t.Errorf("wrong config\ngot:  %s\nwant: %s", got, want)
		}
		// The provider schema includes a 'region' attr, but it's unset in config; null.
		if got, want := normalizeJSON(t, backendState.StateStore.Provider.ConfigRaw), `{"region":null}`; got != want {
			t.Errorf("wrong config\ngot:  %s\nwant: %s", got, want)
		}
	}

	t.Run("provider is already in the dependency lock file", func(t *testing.T) {
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
			"hashicorp/test": {"1.2.3"},
		})

		ui := testUiWrapped(t)
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

		expectedMsg := []string{
			`Migrating state from state store "test_src" (hashicorp/test) to state store "test_dst" (hashicorp/test)...`,
			"Initializing provider plugin for state store \"test_src\"...\n- Reusing previous version of hashicorp/test from the dependency lock file",
			"Initializing provider plugin for state store \"test_dst\"...\n- Reusing previous version of hashicorp/test from the dependency lock file",
		}
		for _, expectedMsg := range expectedMsg {
			if !strings.Contains(out.Stdout(), expectedMsg) {
				t.Fatalf("expected output %q, got %q", expectedMsg, out.Stdout())
			}
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

		// Assert the backend state file describes the new state store location.
		assertBackendStateFile(t, c)
	})

	t.Run("no existing dependency lock file: provider is downloaded and added to the dependency lock file", func(t *testing.T) {
		wd := tempWorkingDirFixture(t, "state-migrate-state-store-to-state-store")
		t.Chdir(wd.RootModuleDir())

		// In this scenario, there is no provider in the dep lock file
		// Achieve this by truncating the dep lock file from the copied fixtures
		if err := os.Truncate(filepath.Join(wd.RootModuleDir(), dependencyLockFilename), 0); err != nil {
			t.Fatalf("error while truncating lock file during test setup: %s", err)
		}

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
			"hashicorp/test": {"1.2.3"},
		})

		ui := testUiWrapped(t)
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

		expectedMsg := []string{
			`Migrating state from state store "test_src" (hashicorp/test) to state store "test_dst" (hashicorp/test)...`,
			"Initializing provider plugin for state store \"test_src\"...\n- Finding hashicorp/test versions matching \"1.2.3\"...\n- Installing hashicorp/test v1.2.3...\n- Installed hashicorp/test v1.2.3 (verified checksum)",
			"Initializing provider plugin for state store \"test_dst\"...\n- Reusing previous version of hashicorp/test from the dependency lock file\n- Using previously-installed hashicorp/test v1.2.3",
		}
		for _, expectedMsg := range expectedMsg {
			if !strings.Contains(out.Stdout(), expectedMsg) {
				t.Fatalf("expected output %q, got %q", expectedMsg, out.Stdout())
			}
		}

		// Assert the state is migrated successfully to the
		// destination state store by inspecting the mock.
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

		// Assert the provider is added to the dependency lock file
		lockFilePath := filepath.Join(wd.RootModuleDir(), dependencyLockFilename)
		lockFileBytes, err := os.ReadFile(lockFilePath)
		if err != nil {
			t.Fatalf("unable to read dependency lock file: %s", err)
		}
		if !strings.Contains(string(lockFileBytes), "hashicorp/test") {
			t.Fatalf("expected provider hashicorp/test to be added to the dependency lock file, got: %s", string(lockFileBytes))
		}

		// Assert the backend state file describes the new state store location.
		assertBackendStateFile(t, c)
	})
}

// Test migration between two state stores in different providers.
// Different cases describe whether the source provider is already in the dependency lock file or not.
func TestStateMigrate_fromStateStoreToStateStore_inDifferentProviders(t *testing.T) {
	// All test cases perform a migration to hashicorp/test2, version v3.2.1, test2_store store.
	// So a common assertion func can be re-used.
	assertBackendStateFile := func(t *testing.T, c *StateMigrateCommand) {
		statePath := filepath.Join(c.DataDir(), DefaultStateFilename)
		sMgr := &clistate.LocalState{Path: statePath}
		if err := sMgr.RefreshState(); err != nil {
			t.Fatal(err)
		}

		backendState := sMgr.State()
		if backendState.Backend != nil {
			t.Fatalf("expected backend state file to not describe a backend during state_store=>state_store migration, but it's set")
		}
		if backendState.StateStore == nil {
			t.Fatalf("expected backend state file to describe a state store during state_store=>state_store migration, but it's nil")
		}
		if backendState.StateStore.Provider.Source.Type != "test2" {
			t.Fatalf("expected backend state file to describe the destination state store provider \"test2\", but got %q", backendState.StateStore.Provider)
		}
		if backendState.StateStore.Provider.Version.String() != "3.2.1" {
			t.Fatalf("expected backend state file to describe the destination state store provider version \"3.2.1\", but got %q", backendState.StateStore.Provider.Version)
		}
		if backendState.StateStore.Type != "test2_dst" {
			t.Fatalf("expected backend state file to describe the destination state store type \"test2_dst\", but got %q", backendState.StateStore.Type)
		}
		// The state store schema is empty, so not even any null fields in expected JSON
		if got, want := normalizeJSON(t, backendState.StateStore.ConfigRaw), `{}`; got != want {
			t.Errorf("wrong config\ngot:  %s\nwant: %s", got, want)
		}
		// The provider schema includes a 'region' attr, but it's unset in config; null.
		if got, want := normalizeJSON(t, backendState.StateStore.Provider.ConfigRaw), `{"region":"foobar"}`; got != want {
			t.Errorf("wrong config\ngot:  %s\nwant: %s", got, want)
		}
	}
	t.Run("source provider already in the dependency lock file, destination is not", func(t *testing.T) {
		fixture := "state-store-changed/provider-used"
		wd := tempWorkingDirFixture(t, fixture)
		t.Chdir(wd.RootModuleDir())

		b, err := os.ReadFile("source-pss.tfstate")
		if err != nil {
			t.Fatal(err)
		}

		// hashicorp/test
		sourcePssSchema := map[string]providers.Schema{
			"test_src": {
				Body: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{},
				},
			},
		}
		sourceProvider := mockPluggableStateStorageProvider(sourcePssSchema)
		sourceProvider.MockStates = testing_provider.MockStateBytes{
			"test_src": map[string][]byte{"default": []byte(b)},
		}
		// hashicorp/test2
		destinationPssSchema := map[string]providers.Schema{
			"test2_dst": {
				Body: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{},
				},
			},
		}
		destinationProvider := mockPluggableStateStorageProvider(destinationPssSchema)
		destinationProvider.MockStates = testing_provider.MockStateBytes{
			"test2_dst": map[string][]byte{}, // No existing state in the destination
		}
		providerSource := newMockProviderSource(t, map[string][]string{
			"hashicorp/test":  {"1.2.3"},
			"hashicorp/test2": {"3.2.1"},
		})

		ui := testUiWrapped(t)
		view, done := testView(t)
		c := &StateMigrateCommand{
			Meta: Meta{
				Ui:                        ui,
				View:                      view,
				WorkingDir:                wd,
				AllowExperimentalFeatures: true,
				testingOverrides: &testingOverrides{
					Providers: map[addrs.Provider]providers.Factory{
						addrs.NewDefaultProvider("test"):  providers.FactoryFixed(sourceProvider),
						addrs.NewDefaultProvider("test2"): providers.FactoryFixed(destinationProvider),
					},
				},
				ProviderSource: providerSource,
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

		// Assert expected human output is made
		// Parameterized due to output referencing the current platform.
		checkParameterizedGoldenReferenceHumanOutput(t, out, fixture, getproviders.CurrentPlatform.String())

		// Assert the state is migrated successfully to the destination state store by inspecting the mock.
		b, err = destinationProvider.MockStates.Read("test2_dst", "default")
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

		// Assert the destination provider is added to the dependency lock file
		// Also, no providers have been removed from the file.
		lockFilePath := filepath.Join(wd.RootModuleDir(), dependencyLockFilename)
		lockFileBytes, err := os.ReadFile(lockFilePath)
		if err != nil {
			t.Fatalf("unable to read dependency lock file: %s", err)
		}
		expectedContents := `# This file is maintained automatically by "terraform init".
# Manual edits may be lost in future updates.

provider "registry.terraform.io/hashicorp/test" {
  version = "1.2.3"
}

provider "registry.terraform.io/hashicorp/test2" {
  version = "3.2.1"
  hashes = [
    "h1:gv1gFnIZulslzchnaoyMJ5KoPvoRgVvSGb3tVS803iw=",
  ]
}
`
		if diff := cmp.Diff(expectedContents, string(lockFileBytes)); diff != "" {
			t.Fatalf("unexpected dependency lock file contents, diff:\n%s", diff)
		}

		// Assert the backend state file describes the new state store location.
		assertBackendStateFile(t, c)
	})
	t.Run("destination provider already in the dependency lock file, source is not", func(t *testing.T) {
		wd := tempWorkingDirFixture(t, "state-store-changed/provider-used")
		t.Chdir(wd.RootModuleDir())

		// Replace dep lock file in fixtures so that the destination provider is already in the dep lock file, but the source provider is not.
		lockFileContents := `# This file is maintained automatically by "terraform init".
# Manual edits may be lost in future updates.

provider "registry.terraform.io/hashicorp/test2" {
  version = "3.2.1"
  hashes = [
    "h1:gv1gFnIZulslzchnaoyMJ5KoPvoRgVvSGb3tVS803iw=",
  ]
}
`
		if err := os.WriteFile(filepath.Join(wd.RootModuleDir(), dependencyLockFilename), []byte(lockFileContents), 0644); err != nil {
			t.Fatalf("unable to overwrite dependency lock file as part of test setup: %s", err)
		}

		b, err := os.ReadFile("source-pss.tfstate")
		if err != nil {
			t.Fatal(err)
		}

		// hashicorp/test
		sourcePssSchema := map[string]providers.Schema{
			"test_src": {
				Body: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{},
				},
			},
		}
		sourceProvider := mockPluggableStateStorageProvider(sourcePssSchema)
		sourceProvider.MockStates = testing_provider.MockStateBytes{
			"test_src": map[string][]byte{"default": []byte(b)},
		}
		// hashicorp/test2
		destinationPssSchema := map[string]providers.Schema{
			"test2_dst": {
				Body: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{},
				},
			},
		}
		destinationProvider := mockPluggableStateStorageProvider(destinationPssSchema)
		destinationProvider.MockStates = testing_provider.MockStateBytes{
			"test2_dst": map[string][]byte{}, // No existing state in the destination
		}
		providerSource := newMockProviderSource(t, map[string][]string{
			"hashicorp/test":  {"1.2.3"},
			"hashicorp/test2": {"3.2.1"},
		})

		ui := testUiWrapped(t)
		view, done := testView(t)
		c := &StateMigrateCommand{
			Meta: Meta{
				Ui:                        ui,
				View:                      view,
				WorkingDir:                wd,
				AllowExperimentalFeatures: true,
				testingOverrides: &testingOverrides{
					Providers: map[addrs.Provider]providers.Factory{
						addrs.NewDefaultProvider("test"):  providers.FactoryFixed(sourceProvider),
						addrs.NewDefaultProvider("test2"): providers.FactoryFixed(destinationProvider),
					},
				},
				ProviderSource: providerSource,
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

		expectedMsg := []string{
			"Initializing provider plugin for state store \"test_src\"...\n- Finding hashicorp/test versions matching \"1.2.3\"...\n- Installing hashicorp/test v1.2.3...\n- Installed hashicorp/test v1.2.3 (verified checksum)",
			"Initializing provider plugin for state store \"test2_dst\"...\n- Reusing previous version of hashicorp/test2 from the dependency lock file",
			`Migrating state from state store "test_src" (hashicorp/test) to state store "test2_dst" (hashicorp/test2)...`,
		}
		for _, expectedMsg := range expectedMsg {
			if !strings.Contains(out.Stdout(), expectedMsg) {
				t.Fatalf("expected output %q, got %q", expectedMsg, out.Stdout())
			}
		}

		// Assert the state is migrated successfully to the destination state store by inspecting the mock.
		b, err = destinationProvider.MockStates.Read("test2_dst", "default")
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

		// Assert the dependency lock file is unchanged, as it was already in the lock file.
		lockFilePath := filepath.Join(wd.RootModuleDir(), dependencyLockFilename)
		lockFileBytes, err := os.ReadFile(lockFilePath)
		if err != nil {
			t.Fatalf("unable to read dependency lock file: %s", err)
		}

		if diff := cmp.Diff(lockFileContents, string(lockFileBytes)); diff != "" {
			t.Fatalf("unexpected dependency lock file contents, diff:\n%s", diff)
		}

		// Assert the backend state file describes the new state store location.
		assertBackendStateFile(t, c)
	})
	t.Run("no existing dependency lock file: only destination provider saved to the dependency lock file", func(t *testing.T) {
		wd := tempWorkingDirFixture(t, "state-store-changed/provider-used")
		t.Chdir(wd.RootModuleDir())

		// Remove lock file, so source provider isn't in the dep lock file for this test scenario
		if err := os.Remove(filepath.Join(wd.RootModuleDir(), dependencyLockFilename)); err != nil {
			t.Fatalf("error while deleting lock file during test setup: %s", err)
		}

		b, err := os.ReadFile("source-pss.tfstate")
		if err != nil {
			t.Fatal(err)
		}

		// hashicorp/test
		sourcePssSchema := map[string]providers.Schema{
			"test_src": {
				Body: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{},
				},
			},
		}
		sourceProvider := mockPluggableStateStorageProvider(sourcePssSchema)
		sourceProvider.MockStates = testing_provider.MockStateBytes{
			"test_src": map[string][]byte{"default": []byte(b)},
		}
		// hashicorp/test2
		destinationPssSchema := map[string]providers.Schema{
			"test2_dst": {
				Body: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{},
				},
			},
		}
		destinationProvider := mockPluggableStateStorageProvider(destinationPssSchema)
		destinationProvider.MockStates = testing_provider.MockStateBytes{
			"test2_dst": map[string][]byte{}, // No existing state in the destination
		}
		providerSource := newMockProviderSource(t, map[string][]string{
			"hashicorp/test":  {"1.2.3"},
			"hashicorp/test2": {"3.2.1"},
		})

		ui := testUiWrapped(t)
		view, done := testView(t)
		c := &StateMigrateCommand{
			Meta: Meta{
				Ui:                        ui,
				View:                      view,
				WorkingDir:                wd,
				AllowExperimentalFeatures: true,
				testingOverrides: &testingOverrides{
					Providers: map[addrs.Provider]providers.Factory{
						addrs.NewDefaultProvider("test"):  providers.FactoryFixed(sourceProvider),
						addrs.NewDefaultProvider("test2"): providers.FactoryFixed(destinationProvider),
					},
				},
				ProviderSource: providerSource,
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

		expectedMsg := []string{
			"Initializing provider plugin for state store \"test_src\"...\n- Finding hashicorp/test versions matching \"1.2.3\"...\n- Installing hashicorp/test v1.2.3...\n- Installed hashicorp/test v1.2.3 (verified checksum)",
			"Initializing provider plugin for state store \"test2_dst\"...\n- Finding latest version of hashicorp/test2...\n- Installing hashicorp/test2 v3.2.1...\n- Installed hashicorp/test2 v3.2.1 (verified checksum)",
			`Migrating state from state store "test_src" (hashicorp/test) to state store "test2_dst" (hashicorp/test2)...`,
		}
		for _, expectedMsg := range expectedMsg {
			if !strings.Contains(out.Stdout(), expectedMsg) {
				t.Fatalf("expected output %q, got %q", expectedMsg, out.Stdout())
			}
		}

		// Assert the state is migrated successfully to the destination state store by inspecting the mock.
		b, err = destinationProvider.MockStates.Read("test2_dst", "default")
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

		// Assert the provider is added to the dependency lock file
		lockFilePath := filepath.Join(wd.RootModuleDir(), dependencyLockFilename)
		lockFileBytes, err := os.ReadFile(lockFilePath)
		if err != nil {
			t.Fatalf("unable to read dependency lock file: %s", err)
		}

		// The source provider is not added here, as it's not used post-migration.
		expectedContents := `# This file is maintained automatically by "terraform init".
# Manual edits may be lost in future updates.

provider "registry.terraform.io/hashicorp/test2" {
  version = "3.2.1"
  hashes = [
    "h1:gv1gFnIZulslzchnaoyMJ5KoPvoRgVvSGb3tVS803iw=",
  ]
}
`
		if diff := cmp.Diff(expectedContents, string(lockFileBytes)); diff != "" {
			t.Fatalf("unexpected dependency lock file contents, diff:\n%s", diff)
		}

		// Assert the backend state file describes the new state store location.
		assertBackendStateFile(t, c)
	})
}

func TestStateMigrate_fromStateStoreToStateStore_interactiveProviderApproval(t *testing.T) {
	t.Run("both providers already in lock file, user not prompted for provider approval", func(t *testing.T) {
		wd := tempWorkingDirFixture(t, "state-store-changed/provider-used")
		t.Chdir(wd.RootModuleDir())

		// Replace dep lock file in fixtures so that both providers are already in the dep lock file.
		lockFileContents := `# This file is maintained automatically by "terraform init".
# Manual edits may be lost in future updates.

provider "registry.terraform.io/hashicorp/test" {
  version = "1.2.3"
}

provider "registry.terraform.io/hashicorp/test2" {
  version = "3.2.1"
  hashes = [
    "h1:gv1gFnIZulslzchnaoyMJ5KoPvoRgVvSGb3tVS803iw=",
  ]
}`
		if err := os.WriteFile(filepath.Join(wd.RootModuleDir(), dependencyLockFilename), []byte(lockFileContents), 0644); err != nil {
			t.Fatalf("unable to overwrite dependency lock file as part of test setup: %s", err)
		}

		b, err := os.ReadFile("source-pss.tfstate")
		if err != nil {
			t.Fatal(err)
		}

		// hashicorp/test - source
		sourcePssSchema := map[string]providers.Schema{
			"test_src": {
				Body: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{},
				},
			},
		}
		sourceProvider := mockPluggableStateStorageProvider(sourcePssSchema)
		sourceProvider.MockStates = testing_provider.MockStateBytes{
			"test_src": map[string][]byte{"default": []byte(b)},
		}
		// hashicorp/test2 - destination
		destinationPssSchema := map[string]providers.Schema{
			"test2_dst": {
				Body: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{},
				},
			},
		}
		destinationProvider := mockPluggableStateStorageProvider(destinationPssSchema)
		destinationProvider.MockStates = testing_provider.MockStateBytes{
			"test2_dst": map[string][]byte{}, // No existing state in the destination
		}

		// Supplying providers via HTTP triggers security features.
		providerSource := newMockProviderSourceUsingTestHttpServer(t, map[string][]string{
			"hashicorp/test":  {"1.2.3"},
			"hashicorp/test2": {"3.2.1"},
		})

		ui := testUiWrapped(t)
		view, done := testView(t)
		c := &StateMigrateCommand{
			Meta: Meta{
				Ui:                        ui,
				View:                      view,
				WorkingDir:                wd,
				AllowExperimentalFeatures: true,
				testingOverrides: &testingOverrides{
					Providers: map[addrs.Provider]providers.Factory{
						addrs.NewDefaultProvider("test"):  providers.FactoryFixed(sourceProvider),
						addrs.NewDefaultProvider("test2"): providers.FactoryFixed(destinationProvider),
					},
				},
				ProviderSource: providerSource,
			},
		}

		_ = testInputMap(t, map[string]string{
			"backend-migrate-copy-to-empty": "yes",
			// Test doesn't assert approval of any providers
		})

		args := []string{"-no-color"}
		code := c.Run(args)
		out := done(t)
		if code != 0 {
			t.Fatalf("unexpected exit: %d\nstderr: %q", code, out.Stderr())
		}

		expectedMsg := []string{
			"Initializing provider plugin for state store \"test_src\"...\n- Reusing previous version of hashicorp/test from the dependency lock file",
			"Initializing provider plugin for state store \"test2_dst\"...\n- Reusing previous version of hashicorp/test2 from the dependency lock file",
			`Migrating state from state store "test_src" (hashicorp/test) to state store "test2_dst" (hashicorp/test2)...`,
		}
		for _, expectedMsg := range expectedMsg {
			if !strings.Contains(out.Stdout(), expectedMsg) {
				t.Fatalf("expected output %q, got %q", expectedMsg, out.Stdout())
			}
		}

		// Assert the state is migrated successfully to the destination state store by inspecting the mock.
		b, err = destinationProvider.MockStates.Read("test2_dst", "default")
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

		// Assert the dependency lock file is unchanged, as both providers were already in the lock file.
		lockFilePath := filepath.Join(wd.RootModuleDir(), dependencyLockFilename)
		lockFileBytes, err := os.ReadFile(lockFilePath)
		if err != nil {
			t.Fatalf("unable to read dependency lock file: %s", err)
		}

		if diff := cmp.Diff(lockFileContents, string(lockFileBytes)); diff != "" {
			t.Fatalf("unexpected dependency lock file contents, diff:\n%s", diff)
		}
	})

	t.Run("only destination provider already in lock file, user prompted once", func(t *testing.T) {
		wd := tempWorkingDirFixture(t, "state-store-changed/provider-used")
		t.Chdir(wd.RootModuleDir())

		// Replace dep lock file in fixtures so that the destination provider is already in the dep lock file, but the source provider is not.
		lockFileContents := `# This file is maintained automatically by "terraform init".
# Manual edits may be lost in future updates.

provider "registry.terraform.io/hashicorp/test2" {
  version = "3.2.1"
  hashes = [
    "h1:gv1gFnIZulslzchnaoyMJ5KoPvoRgVvSGb3tVS803iw=",
  ]
}`
		if err := os.WriteFile(filepath.Join(wd.RootModuleDir(), dependencyLockFilename), []byte(lockFileContents), 0644); err != nil {
			t.Fatalf("unable to overwrite dependency lock file as part of test setup: %s", err)
		}

		b, err := os.ReadFile("source-pss.tfstate")
		if err != nil {
			t.Fatal(err)
		}

		// hashicorp/test
		sourcePssSchema := map[string]providers.Schema{
			"test_src": {
				Body: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{},
				},
			},
		}
		sourceProvider := mockPluggableStateStorageProvider(sourcePssSchema)
		sourceProvider.MockStates = testing_provider.MockStateBytes{
			"test_src": map[string][]byte{"default": []byte(b)},
		}
		// hashicorp/test2
		destinationPssSchema := map[string]providers.Schema{
			"test2_dst": {
				Body: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{},
				},
			},
		}
		destinationProvider := mockPluggableStateStorageProvider(destinationPssSchema)
		destinationProvider.MockStates = testing_provider.MockStateBytes{
			"test2_dst": map[string][]byte{}, // No existing state in the destination
		}

		// Supplying providers via HTTP triggers security features.
		providerSource := newMockProviderSourceUsingTestHttpServer(t, map[string][]string{
			"hashicorp/test":  {"1.2.3"},
			"hashicorp/test2": {"3.2.1"},
		})

		ui := testUiWrapped(t)
		view, done := testView(t)
		c := &StateMigrateCommand{
			Meta: Meta{
				Ui:                        ui,
				View:                      view,
				WorkingDir:                wd,
				AllowExperimentalFeatures: true,
				testingOverrides: &testingOverrides{
					Providers: map[addrs.Provider]providers.Factory{
						addrs.NewDefaultProvider("test"):  providers.FactoryFixed(sourceProvider),
						addrs.NewDefaultProvider("test2"): providers.FactoryFixed(destinationProvider),
					},
				},
				ProviderSource: providerSource,
			},
		}

		_ = testInputMap(t, map[string]string{
			"approve-provider-test-1.2.3":   "yes",
			"backend-migrate-copy-to-empty": "yes",
		})

		args := []string{"-no-color"}
		code := c.Run(args)
		out := done(t)
		if code != 0 {
			t.Fatalf("unexpected exit: %d\nstderr: %q", code, out.Stderr())
		}

		expectedMsg := []string{
			"Initializing provider plugin for state store \"test_src\"...\n- Finding hashicorp/test versions matching \"1.2.3\"...\n- Installing hashicorp/test v1.2.3...\n- Installed hashicorp/test v1.2.3 (verified checksum)",
			"Initializing provider plugin for state store \"test2_dst\"...\n- Reusing previous version of hashicorp/test2 from the dependency lock file",
			`Migrating state from state store "test_src" (hashicorp/test) to state store "test2_dst" (hashicorp/test2)...`,
		}
		for _, expectedMsg := range expectedMsg {
			if !strings.Contains(out.Stdout(), expectedMsg) {
				t.Fatalf("expected output %q, got %q", expectedMsg, out.Stdout())
			}
		}

		// Assert the state is migrated successfully to the destination state store by inspecting the mock.
		b, err = destinationProvider.MockStates.Read("test2_dst", "default")
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

		// Assert the dependency lock file is unchanged, as it was already in the lock file.
		lockFilePath := filepath.Join(wd.RootModuleDir(), dependencyLockFilename)
		lockFileBytes, err := os.ReadFile(lockFilePath)
		if err != nil {
			t.Fatalf("unable to read dependency lock file: %s", err)
		}

		if diff := cmp.Diff(lockFileContents, string(lockFileBytes)); diff != "" {
			t.Fatalf("unexpected dependency lock file contents, diff:\n%s", diff)
		}
	})

	t.Run("only source provider already in lock file, user prompted once", func(t *testing.T) {
		wd := tempWorkingDirFixture(t, "state-store-changed/provider-used")
		t.Chdir(wd.RootModuleDir())

		b, err := os.ReadFile("source-pss.tfstate")
		if err != nil {
			t.Fatal(err)
		}

		// hashicorp/test - source
		sourcePssSchema := map[string]providers.Schema{
			"test_src": {
				Body: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{},
				},
			},
		}
		sourceProvider := mockPluggableStateStorageProvider(sourcePssSchema)
		sourceProvider.MockStates = testing_provider.MockStateBytes{
			"test_src": map[string][]byte{"default": []byte(b)},
		}
		// hashicorp/test2 - destination
		destinationPssSchema := map[string]providers.Schema{
			"test2_dst": {
				Body: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{},
				},
			},
		}
		destinationProvider := mockPluggableStateStorageProvider(destinationPssSchema)
		destinationProvider.MockStates = testing_provider.MockStateBytes{
			"test2_dst": map[string][]byte{}, // No existing state in the destination
		}

		// Supplying providers via HTTP triggers security features.
		providerSource := newMockProviderSourceUsingTestHttpServer(t, map[string][]string{
			"hashicorp/test":  {"1.2.3"},
			"hashicorp/test2": {"3.2.1"},
		})

		ui := testUiWrapped(t)
		view, done := testView(t)
		c := &StateMigrateCommand{
			Meta: Meta{
				Ui:                        ui,
				View:                      view,
				WorkingDir:                wd,
				AllowExperimentalFeatures: true,
				testingOverrides: &testingOverrides{
					Providers: map[addrs.Provider]providers.Factory{
						addrs.NewDefaultProvider("test"):  providers.FactoryFixed(sourceProvider),
						addrs.NewDefaultProvider("test2"): providers.FactoryFixed(destinationProvider),
					},
				},
				ProviderSource: providerSource,
			},
		}

		_ = testInputMap(t, map[string]string{
			"approve-provider-test2-3.2.1":  "yes",
			"backend-migrate-copy-to-empty": "yes",
		})

		args := []string{"-no-color"}
		code := c.Run(args)
		out := done(t)
		if code != 0 {
			t.Fatalf("unexpected exit: %d\nstderr: %q", code, out.Stderr())
		}

		expectedMsg := []string{
			"Initializing provider plugin for state store \"test_src\"...\n- Reusing previous version of hashicorp/test from the dependency lock file",
			"Initializing provider plugin for state store \"test2_dst\"...\n- Finding latest version of hashicorp/test2...\n- Installing hashicorp/test2 v3.2.1...\n- Installed hashicorp/test2 v3.2.1 (verified checksum)",
			`Migrating state from state store "test_src" (hashicorp/test) to state store "test2_dst" (hashicorp/test2)...`,
		}
		for _, expectedMsg := range expectedMsg {
			if !strings.Contains(out.Stdout(), expectedMsg) {
				t.Fatalf("expected output %q, got %q", expectedMsg, out.Stdout())
			}
		}

		// Assert the state is migrated successfully to the destination state store by inspecting the mock.
		b, err = destinationProvider.MockStates.Read("test2_dst", "default")
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

		// Assert the dependency lock file contains the destination provider,
		// and the source provider hasn't been removed.
		lockFilePath := filepath.Join(wd.RootModuleDir(), dependencyLockFilename)
		lockFileBytes, err := os.ReadFile(lockFilePath)
		if err != nil {
			t.Fatalf("unable to read dependency lock file: %s", err)
		}

		lockFileContents := `# This file is maintained automatically by "terraform init".
# Manual edits may be lost in future updates.

provider "registry.terraform.io/hashicorp/test" {
  version = "1.2.3"
}

provider "registry.terraform.io/hashicorp/test2" {
  version = "3.2.1"
  hashes = [
    "h1:gv1gFnIZulslzchnaoyMJ5KoPvoRgVvSGb3tVS803iw=",
  ]
}
`

		if diff := cmp.Diff(lockFileContents, string(lockFileBytes)); diff != "" {
			t.Fatalf("unexpected dependency lock file contents, diff:\n%s", diff)
		}
	})

	t.Run("no lock file, user prompted for both providers", func(t *testing.T) {
		wd := tempWorkingDirFixture(t, "state-store-changed/provider-used")
		t.Chdir(wd.RootModuleDir())

		// Remove lock file, so source provider isn't in the dep lock file for this test scenario
		if err := os.Remove(filepath.Join(wd.RootModuleDir(), dependencyLockFilename)); err != nil {
			t.Fatalf("error while deleting lock file during test setup: %s", err)
		}

		b, err := os.ReadFile("source-pss.tfstate")
		if err != nil {
			t.Fatal(err)
		}

		// hashicorp/test - source
		sourcePssSchema := map[string]providers.Schema{
			"test_src": {
				Body: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{},
				},
			},
		}
		sourceProvider := mockPluggableStateStorageProvider(sourcePssSchema)
		sourceProvider.MockStates = testing_provider.MockStateBytes{
			"test_src": map[string][]byte{"default": []byte(b)},
		}
		// hashicorp/test2 - destination
		destinationPssSchema := map[string]providers.Schema{
			"test2_dst": {
				Body: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{},
				},
			},
		}
		destinationProvider := mockPluggableStateStorageProvider(destinationPssSchema)
		destinationProvider.MockStates = testing_provider.MockStateBytes{
			"test2_dst": map[string][]byte{}, // No existing state in the destination
		}

		// Supplying providers via HTTP triggers security features.
		providerSource := newMockProviderSourceUsingTestHttpServer(t, map[string][]string{
			"hashicorp/test":  {"1.2.3"},
			"hashicorp/test2": {"3.2.1"},
		})

		ui := cli.NewMockUi()
		view, done := testView(t)
		c := &StateMigrateCommand{
			Meta: Meta{
				Ui:                        ui,
				View:                      view,
				WorkingDir:                wd,
				AllowExperimentalFeatures: true,
				testingOverrides: &testingOverrides{
					Providers: map[addrs.Provider]providers.Factory{
						addrs.NewDefaultProvider("test"):  providers.FactoryFixed(sourceProvider),
						addrs.NewDefaultProvider("test2"): providers.FactoryFixed(destinationProvider),
					},
				},
				ProviderSource: providerSource,
			},
		}

		_ = testInputMap(t, map[string]string{
			"approve-provider-test-1.2.3":   "yes",
			"approve-provider-test2-3.2.1":  "yes",
			"backend-migrate-copy-to-empty": "yes",
		})

		args := []string{"-no-color"}
		code := c.Run(args)
		out := done(t)
		if code != 0 {
			t.Fatalf("unexpected exit: %d\nstderr: %q", code, out.Stderr())
		}

		expectedMsg := []string{
			"Initializing provider plugin for state store \"test_src\"...\n- Finding hashicorp/test versions matching \"1.2.3\"...\n- Installing hashicorp/test v1.2.3...\n- Installed hashicorp/test v1.2.3 (verified checksum)",
			"Initializing provider plugin for state store \"test2_dst\"...\n- Finding latest version of hashicorp/test2...\n- Installing hashicorp/test2 v3.2.1...\n- Installed hashicorp/test2 v3.2.1 (verified checksum)",
			`Migrating state from state store "test_src" (hashicorp/test) to state store "test2_dst" (hashicorp/test2)...`,
		}
		for _, expectedMsg := range expectedMsg {
			if !strings.Contains(out.Stdout(), expectedMsg) {
				t.Fatalf("expected output %q, got %q", expectedMsg, out.Stdout())
			}
		}

		// Assert the state is migrated successfully to the destination state store by inspecting the mock.
		b, err = destinationProvider.MockStates.Read("test2_dst", "default")
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

		// Assert the dependency lock file contains the destination provider,
		// and the source provider hasn't been removed.
		lockFilePath := filepath.Join(wd.RootModuleDir(), dependencyLockFilename)
		lockFileBytes, err := os.ReadFile(lockFilePath)
		if err != nil {
			t.Fatalf("unable to read dependency lock file: %s", err)
		}

		lockFileContents := `# This file is maintained automatically by "terraform init".
# Manual edits may be lost in future updates.

provider "registry.terraform.io/hashicorp/test2" {
  version = "3.2.1"
  hashes = [
    "h1:gv1gFnIZulslzchnaoyMJ5KoPvoRgVvSGb3tVS803iw=",
  ]
}
`

		if diff := cmp.Diff(lockFileContents, string(lockFileBytes)); diff != "" {
			t.Fatalf("unexpected dependency lock file contents, diff:\n%s", diff)
		}
	})
}

func TestStateMigrate_stateStore_newWorkingDir_inAutomationProviderApproval(t *testing.T) {
	t.Run("both providers already in lock file, user not prompted for provider approval", func(t *testing.T) {
		wd := tempWorkingDirFixture(t, "state-store-changed/provider-used")
		t.Chdir(wd.RootModuleDir())

		// Replace dep lock file in fixtures so that both providers are already in the dep lock file.
		lockFileContents := `# This file is maintained automatically by "terraform init".
# Manual edits may be lost in future updates.

provider "registry.terraform.io/hashicorp/test" {
  version = "1.2.3"
}

provider "registry.terraform.io/hashicorp/test2" {
  version = "3.2.1"
  hashes = [
    "h1:gv1gFnIZulslzchnaoyMJ5KoPvoRgVvSGb3tVS803iw=",
  ]
}`
		if err := os.WriteFile(filepath.Join(wd.RootModuleDir(), dependencyLockFilename), []byte(lockFileContents), 0644); err != nil {
			t.Fatalf("unable to overwrite dependency lock file as part of test setup: %s", err)
		}

		b, err := os.ReadFile("source-pss.tfstate")
		if err != nil {
			t.Fatal(err)
		}

		// hashicorp/test - source
		sourcePssSchema := map[string]providers.Schema{
			"test_src": {
				Body: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{},
				},
			},
		}
		sourceProvider := mockPluggableStateStorageProvider(sourcePssSchema)
		sourceProvider.MockStates = testing_provider.MockStateBytes{
			"test_src": map[string][]byte{"default": []byte(b)},
		}
		// hashicorp/test2 - destination
		destinationPssSchema := map[string]providers.Schema{
			"test2_dst": {
				Body: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{},
				},
			},
		}
		destinationProvider := mockPluggableStateStorageProvider(destinationPssSchema)
		destinationProvider.MockStates = testing_provider.MockStateBytes{
			"test2_dst": map[string][]byte{}, // No existing state in the destination
		}

		// Supplying providers via HTTP triggers security features.
		providerSource := newMockProviderSourceUsingTestHttpServer(t, map[string][]string{
			"hashicorp/test":  {"1.2.3"},
			"hashicorp/test2": {"3.2.1"},
		})

		ui := testUiWrapped(t)
		view, done := testView(t)
		c := &StateMigrateCommand{
			Meta: Meta{
				Ui:                        ui,
				View:                      view,
				WorkingDir:                wd,
				AllowExperimentalFeatures: true,
				testingOverrides: &testingOverrides{
					Providers: map[addrs.Provider]providers.Factory{
						addrs.NewDefaultProvider("test"):  providers.FactoryFixed(sourceProvider),
						addrs.NewDefaultProvider("test2"): providers.FactoryFixed(destinationProvider),
					},
				},
				ProviderSource: providerSource,
			},
		}

		args := []string{
			"-input=false", // Simulate running in automation where input is disabled
			"-force-copy",  // Suppress prompts to perform the migration
			"-no-color",
		}
		code := c.Run(args)
		out := done(t)
		if code != 0 {
			t.Fatalf("unexpected exit: %d\nstderr: %q", code, out.Stderr())
		}

		expectedMsg := []string{
			"Initializing provider plugin for state store \"test_src\"...\n- Reusing previous version of hashicorp/test from the dependency lock file",
			"Initializing provider plugin for state store \"test2_dst\"...\n- Reusing previous version of hashicorp/test2 from the dependency lock file",
			`Migrating state from state store "test_src" (hashicorp/test) to state store "test2_dst" (hashicorp/test2)...`,
		}
		for _, expectedMsg := range expectedMsg {
			if !strings.Contains(out.Stdout(), expectedMsg) {
				t.Fatalf("expected output %q, got %q", expectedMsg, out.Stdout())
			}
		}
		// notExpectedMsg := []string{
		// 	"The state store provider was approved automatically", // shouldn't be there as no explicit CLI flag used.
		// }
		// for _, notExpected := range notExpectedMsg {
		// 	if strings.Contains(out.Stdout(), notExpected) {
		// 		t.Fatalf("did not expect output %q, but got %q", notExpected, out.Stdout())
		// 	}
		// }

		// Assert the state is migrated successfully to the destination state store by inspecting the mock.
		b, err = destinationProvider.MockStates.Read("test2_dst", "default")
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

		// Assert the dependency lock file is unchanged, as both providers were already in the lock file.
		lockFilePath := filepath.Join(wd.RootModuleDir(), dependencyLockFilename)
		lockFileBytes, err := os.ReadFile(lockFilePath)
		if err != nil {
			t.Fatalf("unable to read dependency lock file: %s", err)
		}

		if diff := cmp.Diff(lockFileContents, string(lockFileBytes)); diff != "" {
			t.Fatalf("unexpected dependency lock file contents, diff:\n%s", diff)
		}
	})

	t.Run("users approve trusting a state store provider downloaded via HTTP by supplying locks via -source-provider-lock-file and -destination-provider-lock-file flags", func(t *testing.T) {
		wd := tempWorkingDirFixture(t, "state-store-changed/provider-used")
		t.Chdir(wd.RootModuleDir())

		b, err := os.ReadFile("source-pss.tfstate")
		if err != nil {
			t.Fatal(err)
		}

		// hashicorp/test - source
		sourcePssSchema := map[string]providers.Schema{
			"test_src": {
				Body: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{},
				},
			},
		}
		srcMockProviderAddress := addrs.NewDefaultProvider("test")
		sourceProvider := mockPluggableStateStorageProvider(sourcePssSchema)
		sourceProvider.MockStates = testing_provider.MockStateBytes{
			"test_src": map[string][]byte{"default": []byte(b)},
		}
		// hashicorp/test2 - destination
		dstMockProviderAddress := addrs.NewDefaultProvider("test2")
		destinationPssSchema := map[string]providers.Schema{
			"test2_dst": {
				Body: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{},
				},
			},
		}
		destinationProvider := mockPluggableStateStorageProvider(destinationPssSchema)
		destinationProvider.MockStates = testing_provider.MockStateBytes{
			"test2_dst": map[string][]byte{}, // No existing state in the destination
		}

		// Set up mock provider source that mocks out hashicorp/test and hashicorp/test2 download via HTTP.
		// This stops Terraform auto-approving the provider installation.
		expectedVersion := "1.2.3"
		source := newMockProviderSourceUsingTestHttpServer(t, map[string][]string{
			"hashicorp/test":  {expectedVersion, "9.9.9"}, // Extra version - expected version is downloaded, not the latest
			"hashicorp/test2": {expectedVersion, "9.9.9"}, // Extra version - expected version is downloaded, not the latest
		})

		ui := new(cli.MockUi)
		view, done := testView(t)
		meta := Meta{
			Ui:                        ui,
			View:                      view,
			AllowExperimentalFeatures: true,
			testingOverrides: &testingOverrides{
				Providers: map[addrs.Provider]providers.Factory{
					srcMockProviderAddress: providers.FactoryFixed(sourceProvider),
					dstMockProviderAddress: providers.FactoryFixed(destinationProvider),
				},
			},
			ProviderSource: source,
			WorkingDir:     wd,
		}
		c := &StateMigrateCommand{
			Meta: meta,
		}

		// Create supplemental lock file to be used with the -source-provider-lock-file
		// and -destination-provider-lock-file flags.
		// To avoid this being confused with the lock file in the working directory,
		// this is made in a second temp directory away from other files in this test.
		td2 := t.TempDir()
		lockFileName := filepath.Join(td2, ".terraform.lock.hcl")
		suppliedLockFileVersion := getproviders.MustParseVersion(expectedVersion)
		locks := depsfile.NewLocks()
		locks.SetProvider(
			srcMockProviderAddress,
			suppliedLockFileVersion,
			getproviders.MustParseVersionConstraints("> 1.0.0"),
			[]getproviders.Hash{
				getproviders.HashScheme1.New("wlbEC2mChQZ2hhgUhl6SeVLPP7fMqOFUZAQhQ9GIIno="),
			},
		)
		locks.SetProvider(
			dstMockProviderAddress,
			suppliedLockFileVersion,
			getproviders.MustParseVersionConstraints("> 1.0.0"),
			[]getproviders.Hash{
				getproviders.HashScheme1.New("Ro/5F9A1n2BADO4uPWVdJpiydN7bMRQOLzlOfb7m04s="),
			},
		)
		depsfile.SaveLocksToFile(locks, lockFileName)

		args := []string{
			"-input=false", // Simulate running in automation where input is disabled
			"-force-copy",  // Suppress prompts to perform the migration
			"-no-color",
			fmt.Sprintf("-source-provider-lock-file=%s", lockFileName),
			fmt.Sprintf("-destination-provider-lock-file=%s", lockFileName),
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 0 {
			t.Fatalf("expected code 0 exit code, got %d, output: \n%s", code, testOutput.All())
		}

		// Check output via view
		output := testOutput.All()
		expectedOutputs := []string{
			"Initializing provider plugin for state store \"test_src\"...\n- Reusing previous version of hashicorp/test from the dependency lock file\n- Installing hashicorp/test v1.2.3...\n- Installed hashicorp/test v1.2.3 (verified checksum)",
			"Initializing provider plugin for state store \"test2_dst\"...\n- Reusing previous version of hashicorp/test2 from the dependency lock file\n- Installing hashicorp/test2 v1.2.3...\n- Installed hashicorp/test2 v1.2.3 (verified checksum)",
			"The state store provider was approved automatically.",
			"Migrating state from state store \"test_src\" (hashicorp/test) to state store \"test2_dst\" (hashicorp/test2)...",
		}
		for _, expected := range expectedOutputs {
			if !strings.Contains(output, expected) {
				t.Fatalf("expected output to include %q, but got':\n %s", expected, output)
			}
		}

		// Assert the dependency lock file was created
		// and it contains the state store provider version described by the -state-provider-lock-file flag
		lockFile := filepath.Join(wd.RootModuleDir(), ".terraform.lock.hcl")
		locks, lockDiags := depsfile.LoadLocksFromFile(lockFile)
		if lockDiags.HasErrors() {
			t.Fatal("expected dependency lock file to exist, but got errors loading it:", lockDiags.Err())
		}
		gotLock := locks.Provider(srcMockProviderAddress)
		if gotLock == nil {
			t.Fatalf("expected dependency lock file to contain the state store provider %s, but it doesn't", srcMockProviderAddress.ForDisplay())
		}
		if !gotLock.Version().Same(suppliedLockFileVersion) {
			t.Fatalf("expected dependency lock file to contain version %s for provider %s that was supplied via the -source-provider-lock-file flag, but got version %s", suppliedLockFileVersion, srcMockProviderAddress.ForDisplay(), gotLock.Version())
		}
		gotLock = locks.Provider(dstMockProviderAddress)
		if gotLock == nil {
			t.Fatalf("expected dependency lock file to contain the state store provider %s, but it doesn't", dstMockProviderAddress.ForDisplay())
		}
		if !gotLock.Version().Same(suppliedLockFileVersion) {
			t.Fatalf("expected dependency lock file to contain version %s for provider %s that was supplied via the -destination-provider-lock-file flag, but got version %s", suppliedLockFileVersion, dstMockProviderAddress.ForDisplay(), gotLock.Version())
		}
	})

	t.Run("error if the lock file supplied by the -source-provider-lock-file flag doesn't contain the state store provider", func(t *testing.T) {
		wd := tempWorkingDirFixture(t, "state-store-changed/provider-used")
		t.Chdir(wd.RootModuleDir())

		b, err := os.ReadFile("source-pss.tfstate")
		if err != nil {
			t.Fatal(err)
		}

		// hashicorp/test - source
		sourcePssSchema := map[string]providers.Schema{
			"test_src": {
				Body: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{},
				},
			},
		}
		srcMockProviderAddress := addrs.NewDefaultProvider("test")
		sourceProvider := mockPluggableStateStorageProvider(sourcePssSchema)
		sourceProvider.MockStates = testing_provider.MockStateBytes{
			"test_src": map[string][]byte{"default": []byte(b)},
		}
		// hashicorp/test2 - destination
		dstMockProviderAddress := addrs.NewDefaultProvider("test2")
		destinationPssSchema := map[string]providers.Schema{
			"test2_dst": {
				Body: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{},
				},
			},
		}
		destinationProvider := mockPluggableStateStorageProvider(destinationPssSchema)
		destinationProvider.MockStates = testing_provider.MockStateBytes{
			"test2_dst": map[string][]byte{}, // No existing state in the destination
		}

		// Set up mock provider source that mocks out hashicorp/test and hashicorp/test2 download via HTTP.
		// This stops Terraform auto-approving the provider installation.
		expectedVersion := "1.2.3"
		source := newMockProviderSourceUsingTestHttpServer(t, map[string][]string{
			"hashicorp/test":  {expectedVersion, "9.9.9"}, // Extra version - expected version is downloaded, not the latest
			"hashicorp/test2": {expectedVersion, "9.9.9"}, // Extra version - expected version is downloaded, not the latest
		})

		ui := new(cli.MockUi)
		view, done := testView(t)
		meta := Meta{
			Ui:                        ui,
			View:                      view,
			AllowExperimentalFeatures: true,
			testingOverrides: &testingOverrides{
				Providers: map[addrs.Provider]providers.Factory{
					srcMockProviderAddress: providers.FactoryFixed(sourceProvider),
					dstMockProviderAddress: providers.FactoryFixed(destinationProvider),
				},
			},
			ProviderSource: source,
			WorkingDir:     wd,
		}
		c := &StateMigrateCommand{
			Meta: meta,
		}

		// Create supplemental lock file to be used with the -source-provider-lock-file flag
		// To avoid this being confused with the lock file in the working directory,
		// this is made in a second temp directory away from other files in this test.
		td2 := t.TempDir()
		lockFileName := filepath.Join(td2, ".terraform.lock.hcl")

		// It DOESNT contain the state store provider hashicorp/test though, causing an error.
		locks := depsfile.NewLocks()
		locks.SetProvider(
			addrs.NewDefaultProvider("notusedprovider"),
			getproviders.MustParseVersion("9.9.9"),
			getproviders.MustParseVersionConstraints("> 1.0.0"),
			[]getproviders.Hash{
				getproviders.HashScheme1.New("wlbEC2mChQZ2hhgUhl6SeVLPP7fMqOFUZAQhQ9GIIno="),
			},
		)
		depsfile.SaveLocksToFile(locks, lockFileName)

		args := []string{
			"-input=false", // Simulate running in automation where input is disabled
			"-force-copy",  // Suppress prompts to perform the migration
			"-no-color",
			fmt.Sprintf("-source-provider-lock-file=%s", lockFileName),
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 1 {
			t.Fatalf("expected code 1 exit code, got %d, output: \n%s", code, testOutput.All())
		}

		// Check output via view
		output := cleanString(testOutput.All())
		expectedOutputs := []string{
			"Error: Missing lock for state store provider",
			fmt.Sprintf(`The lock file at %q (supplied via CLI flag) was empty or did not contain a lock for the state store provider.`, lockFileName),
		}
		for _, expected := range expectedOutputs {
			if !strings.Contains(output, expected) {
				t.Fatalf("expected output to include %q, but got':\n %s", expected, output)
			}
		}
	})

	t.Run("error if the lock file supplied by the -destination-provider-lock-file flag doesn't contain the state store provider", func(t *testing.T) {
		wd := tempWorkingDirFixture(t, "state-store-changed/provider-used")
		t.Chdir(wd.RootModuleDir())

		b, err := os.ReadFile("source-pss.tfstate")
		if err != nil {
			t.Fatal(err)
		}

		// hashicorp/test - source
		sourcePssSchema := map[string]providers.Schema{
			"test_src": {
				Body: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{},
				},
			},
		}
		srcMockProviderAddress := addrs.NewDefaultProvider("test")
		sourceProvider := mockPluggableStateStorageProvider(sourcePssSchema)
		sourceProvider.MockStates = testing_provider.MockStateBytes{
			"test_src": map[string][]byte{"default": []byte(b)},
		}
		// hashicorp/test2 - destination
		dstMockProviderAddress := addrs.NewDefaultProvider("test2")
		destinationPssSchema := map[string]providers.Schema{
			"test2_dst": {
				Body: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{},
				},
			},
		}
		destinationProvider := mockPluggableStateStorageProvider(destinationPssSchema)
		destinationProvider.MockStates = testing_provider.MockStateBytes{
			"test2_dst": map[string][]byte{}, // No existing state in the destination
		}

		// Set up mock provider source that mocks out hashicorp/test and hashicorp/test2 download via HTTP.
		// This stops Terraform auto-approving the provider installation.
		expectedVersion := "1.2.3"
		source := newMockProviderSourceUsingTestHttpServer(t, map[string][]string{
			"hashicorp/test":  {expectedVersion, "9.9.9"}, // Extra version - expected version is downloaded, not the latest
			"hashicorp/test2": {expectedVersion, "9.9.9"}, // Extra version - expected version is downloaded, not the latest
		})

		ui := new(cli.MockUi)
		view, done := testView(t)
		meta := Meta{
			Ui:                        ui,
			View:                      view,
			AllowExperimentalFeatures: true,
			testingOverrides: &testingOverrides{
				Providers: map[addrs.Provider]providers.Factory{
					srcMockProviderAddress: providers.FactoryFixed(sourceProvider),
					dstMockProviderAddress: providers.FactoryFixed(destinationProvider),
				},
			},
			ProviderSource: source,
			WorkingDir:     wd,
		}
		c := &StateMigrateCommand{
			Meta: meta,
		}

		// Create supplemental lock file to be used with the -destination-provider-lock-file flag
		// To avoid this being confused with the lock file in the working directory,
		// this is made in a second temp directory away from other files in this test.
		td2 := t.TempDir()
		lockFileName := filepath.Join(td2, ".terraform.lock.hcl")

		// It DOESNT contain the state store provider hashicorp/test though, causing an error.
		locks := depsfile.NewLocks()
		locks.SetProvider(
			addrs.NewDefaultProvider("notusedprovider"),
			getproviders.MustParseVersion("9.9.9"),
			getproviders.MustParseVersionConstraints("> 1.0.0"),
			[]getproviders.Hash{
				getproviders.HashScheme1.New("wlbEC2mChQZ2hhgUhl6SeVLPP7fMqOFUZAQhQ9GIIno="),
			},
		)
		depsfile.SaveLocksToFile(locks, lockFileName)

		args := []string{
			"-input=false", // Simulate running in automation where input is disabled
			"-force-copy",  // Suppress prompts to perform the migration
			"-no-color",
			// -source-provider-lock-file not set; use default lock file that contains a lock for the old provider used
			fmt.Sprintf("-destination-provider-lock-file=%s", lockFileName),
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 1 {
			t.Fatalf("expected code 1 exit code, got %d, output: \n%s", code, testOutput.All())
		}

		// Check output via view
		output := cleanString(testOutput.All())
		expectedOutputs := []string{
			"Error: Missing lock for state store provider",
			fmt.Sprintf(`The lock file at %q (supplied via CLI flag) was empty or did not contain a lock for the state store provider.`, lockFileName),
		}
		for _, expected := range expectedOutputs {
			if !strings.Contains(output, expected) {
				t.Fatalf("expected output to include %q, but got':\n %s", expected, output)
			}
		}
	})

	t.Run("error if the state store lock is supplied by neither a pre-existing lock nor the -source-provider-lock-file flag", func(t *testing.T) {
		wd := tempWorkingDirFixture(t, "state-store-changed/provider-used")
		t.Chdir(wd.RootModuleDir())

		// Truncate dep lock file in fixtures so no provider are already in the dep lock file.
		if err := os.Truncate(filepath.Join(wd.RootModuleDir(), dependencyLockFilename), 0); err != nil {
			t.Fatalf("unable to overwrite dependency lock file as part of test setup: %s", err)
		}

		b, err := os.ReadFile("source-pss.tfstate")
		if err != nil {
			t.Fatal(err)
		}

		// hashicorp/test - source
		sourcePssSchema := map[string]providers.Schema{
			"test_src": {
				Body: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{},
				},
			},
		}
		srcMockProviderAddress := addrs.NewDefaultProvider("test")
		sourceProvider := mockPluggableStateStorageProvider(sourcePssSchema)
		sourceProvider.MockStates = testing_provider.MockStateBytes{
			"test_src": map[string][]byte{"default": []byte(b)},
		}
		// hashicorp/test2 - destination
		dstMockProviderAddress := addrs.NewDefaultProvider("test2")
		destinationPssSchema := map[string]providers.Schema{
			"test2_dst": {
				Body: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{},
				},
			},
		}
		destinationProvider := mockPluggableStateStorageProvider(destinationPssSchema)
		destinationProvider.MockStates = testing_provider.MockStateBytes{
			"test2_dst": map[string][]byte{}, // No existing state in the destination
		}

		// Set up mock provider source that mocks out hashicorp/test and hashicorp/test2 download via HTTP.
		// This stops Terraform auto-approving the provider installation.
		expectedVersion := "1.2.3"
		source := newMockProviderSourceUsingTestHttpServer(t, map[string][]string{
			"hashicorp/test":  {expectedVersion, "9.9.9"}, // Extra version - expected version is downloaded, not the latest
			"hashicorp/test2": {expectedVersion, "9.9.9"}, // Extra version - expected version is downloaded, not the latest
		})

		ui := new(cli.MockUi)
		view, done := testView(t)
		meta := Meta{
			Ui:                        ui,
			View:                      view,
			AllowExperimentalFeatures: true,
			testingOverrides: &testingOverrides{
				Providers: map[addrs.Provider]providers.Factory{
					srcMockProviderAddress: providers.FactoryFixed(sourceProvider),
					dstMockProviderAddress: providers.FactoryFixed(destinationProvider),
				},
			},
			ProviderSource: source,
			WorkingDir:     wd,
		}
		c := &StateMigrateCommand{
			Meta: meta,
		}

		// Create supplemental lock file to be used with the -destination-provider-lock-file flag
		// To avoid this being confused with the lock file in the working directory,
		// this is made in a second temp directory away from other files in this test.
		td2 := t.TempDir()
		lockFileName := filepath.Join(td2, ".terraform.lock.hcl")

		// It DOESNT contain the state store provider hashicorp/test though, causing an error.
		locks := depsfile.NewLocks()
		locks.SetProvider(
			addrs.NewDefaultProvider("notusedprovider"),
			getproviders.MustParseVersion("9.9.9"),
			getproviders.MustParseVersionConstraints("> 1.0.0"),
			[]getproviders.Hash{
				getproviders.HashScheme1.New("wlbEC2mChQZ2hhgUhl6SeVLPP7fMqOFUZAQhQ9GIIno="),
			},
		)
		depsfile.SaveLocksToFile(locks, lockFileName)

		args := []string{
			"-input=false", // Simulate running in automation where input is disabled
			"-force-copy",  // Suppress prompts to perform the migration
			"-no-color",
			// -source-provider-lock-file not set; use default lock file that contains a lock for the old provider used
			fmt.Sprintf("-destination-provider-lock-file=%s", lockFileName),
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 1 {
			t.Fatalf("expected code 1 exit code, got %d, output: \n%s", code, testOutput.All())
		}

		// Check output via view
		output := cleanString(testOutput.All())
		expectedOutputs := []string{
			"Error: Missing lock for state store provider",
			"Terraform is initializing a state store for the first time in a non-interactive mode but no lock was found for the state store provider. Terraform used the working directory's lock file by default, but it was empty or did not exist.",
		}
		for _, expected := range expectedOutputs {
			if !strings.Contains(output, expected) {
				t.Fatalf("expected output to include %q, but got':\n %s", expected, output)
			}
		}
	})
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
		"hashicorp/test": {"1.2.3"},
	})

	ui := testUiWrapped(t)
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

	expectedMsg := []string{
		"Initializing provider plugin for state store \"test_store\"...\n- Reusing previous version of hashicorp/test from the dependency lock file",
		`Migrating state from state store "test_store" (hashicorp/test) to backend "local"...`,
	}
	for _, expectedMsg := range expectedMsg {
		if !strings.Contains(out.Stdout(), expectedMsg) {
			t.Fatalf("expected output %q, got %q", expectedMsg, out.Stdout())
		}
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

	// Assert the backend state file describes the new backend location,
	statePath := filepath.Join(c.DataDir(), DefaultStateFilename)
	sMgr := &clistate.LocalState{Path: statePath}
	if err := sMgr.RefreshState(); err != nil {
		t.Fatal(err)
	}
	backendState := sMgr.State()
	if backendState.StateStore != nil {
		t.Fatalf("expected backend state file to not describe a state_store during state_store=>backend migration, but it's set")
	}
	if backendState.Backend == nil {
		t.Fatalf("expected backend state file to describe a backend during state_store=>backend migration, but it's nil")
	}
	if backendState.Backend.Type != "local" {
		t.Fatalf("expected backend state file to describe the destination backend \"local\", but got %q", backendState.Backend.Type)
	}
	if got, want := normalizeJSON(t, backendState.Backend.ConfigRaw), `{"path":"destination-backend.tfstate","workspace_dir":null}`; got != want {
		t.Errorf("wrong config\ngot:  %s\nwant: %s", got, want)
	}
}

func TestStateMigrate_missingModuleFiles(t *testing.T) {
	wd := tempWorkingDirFixture(t, "state-migrate-missing-mod-files")
	t.Chdir(wd.RootModuleDir())

	providerSource := newMockProviderSource(t, map[string][]string{
		"hashicorp/test": {"1.2.3"},
	})

	ui := testUiWrapped(t)
	view, done := testView(t)
	c := &StateMigrateCommand{
		Meta: Meta{
			Ui:                        ui,
			View:                      view,
			WorkingDir:                wd,
			AllowExperimentalFeatures: true,
			ProviderSource:            providerSource,
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

	providerSource := newMockProviderSource(t, map[string][]string{
		"hashicorp/test": {"1.2.3"},
	})

	ui := testUiWrapped(t)
	view, done := testView(t)
	c := &StateMigrateCommand{
		Meta: Meta{
			Ui:                        ui,
			View:                      view,
			WorkingDir:                wd,
			AllowExperimentalFeatures: true,
			ProviderSource:            providerSource,
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

	ui := testUiWrapped(t)
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

	ui := testUiWrapped(t)
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
