// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	version "github.com/hashicorp/go-version"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/backend"
	backendInit "github.com/hashicorp/terraform/internal/backend/init"
	httpBackend "github.com/hashicorp/terraform/internal/backend/remote-state/http"
	"github.com/hashicorp/terraform/internal/backend/remote-state/inmem"
	"github.com/hashicorp/terraform/internal/cloud"
	"github.com/hashicorp/terraform/internal/command/clistate"
	"github.com/hashicorp/terraform/internal/command/workdir"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/getproviders"
	"github.com/hashicorp/terraform/internal/providercache"
	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/statefile"
	"github.com/zclconf/go-cty/cty"
)

// Testing init's basic behaviors with `state_store` when run in an empty working directory: backend state file creation, behaviour based on the selected workspace
func TestInit_stateStore_newWorkingDir_basic(t *testing.T) {
	t.Run("the init command creates a backend state file, and the default workspace is not made by default", func(t *testing.T) {
		// Create a temporary, uninitialized working directory with configuration including a state store
		td := t.TempDir()
		testCopyDir(t, testFixturePath("init-with-state-store"), td)
		t.Chdir(td)

		mockProvider := mockPluggableStateStorageProvider(mockSingleStateStoreSchema("test_store"))
		mockProviderAddress := addrs.NewDefaultProvider("test")

		providerSource := newMockProviderSource(t, map[string][]string{
			// The test fixture config has no version constraints, so the latest version will
			// be used; below is the 'latest' version in the test world.
			"hashicorp/test": {"1.2.3"},
		})

		ui := testUiWrapped(t)
		view, done := testView(t)
		meta := Meta{
			Ui:                        ui,
			View:                      view,
			AllowExperimentalFeatures: true,
			testingOverrides: &testingOverrides{
				Providers: map[addrs.Provider]providers.Factory{
					mockProviderAddress: providers.FactoryFixed(mockProvider),
				},
			},
			ProviderSource: providerSource,
		}
		c := &InitCommand{
			Meta: meta,
		}

		args := []string{"-enable-pluggable-state-storage-experiment=true"}
		code := c.Run(args)
		testOutput := done(t)
		if code != 0 {
			t.Fatalf("expected code 0 exit code, got %d, output: \n%s", code, testOutput.All())
		}

		// Check output
		output := testOutput.All()
		expectedOutput := `Installing provider hashicorp/test for state store "test_store"...
- Finding latest version of hashicorp/test...
- Installing hashicorp/test v1.2.3...
- Installed hashicorp/test v1.2.3 (verified checksum)

Initializing the state store "test_store"...

Initializing provider plugins...
- Reusing version 1.2.3 of hashicorp/test from the dependency lock file
- Using previously-installed hashicorp/test v1.2.3`
		if !strings.Contains(output, expectedOutput) {
			t.Fatalf("expected output to include %q, but got:\n%s", expectedOutput, output)
		}

		// Assert the default workspace was not created
		exists, err := mockProvider.MockStates.StateIdExists("test_store", backend.DefaultStateName)
		if err != nil {
			t.Fatal(err)
		}
		if exists {
			t.Fatal("expected the default workspace to not be created during init, but it exists")
		}

		// Assert contents of the backend state file
		statePath := filepath.Join(meta.DataDir(), DefaultStateFilename)
		sMgr := &clistate.LocalState{Path: statePath}
		if err := sMgr.RefreshState(); err != nil {
			t.Fatal("Failed to load state:", err)
		}
		s := sMgr.State()
		if s == nil {
			t.Fatal("expected backend state file to be created, but there isn't one")
		}
		v1_2_3, _ := version.NewVersion("1.2.3")
		expectedState := &workdir.StateStoreConfigState{
			Type:               "test_store",
			ConfigRaw:          []byte("{\n      \"value\": \"foobar\"\n    }"),
			Hash:               uint64(4158988729),
			ProviderSupplyMode: getproviders.ManagedByTerraform,
			Provider: &workdir.ProviderConfigState{
				Version: v1_2_3,
				Source: &tfaddr.Provider{
					Hostname:  tfaddr.DefaultProviderRegistryHost,
					Namespace: "hashicorp",
					Type:      "test",
				},
				ConfigRaw: []byte("{\n        \"region\": null\n      }"),
			},
		}
		if diff := cmp.Diff(s.StateStore, expectedState); diff != "" {
			t.Fatalf("unexpected diff in backend state file's description of state store:\n%s", diff)
		}
	})

	// This scenario would be rare, but protecting against it is easy and avoids assumptions.
	t.Run("error when a custom workspace is selected but no workspaces exist", func(t *testing.T) {
		// Create a temporary, uninitialized working directory with configuration including a state store
		td := t.TempDir()
		testCopyDir(t, testFixturePath("init-with-state-store"), td)
		t.Chdir(td)

		// Select a custom workspace (which will not exist)
		customWorkspace := "my-custom-workspace"
		t.Setenv(WorkspaceNameEnvVar, customWorkspace)

		mockProvider := mockPluggableStateStorageProvider(mockSingleStateStoreSchema("test_store"))
		mockProviderAddress := addrs.NewDefaultProvider("test")
		providerSource := newMockProviderSource(t, map[string][]string{
			"hashicorp/test": {"1.0.0"},
		})

		ui := testUiWrapped(t)
		view, done := testView(t)
		meta := Meta{
			Ui:                        ui,
			View:                      view,
			AllowExperimentalFeatures: true,
			testingOverrides: &testingOverrides{
				Providers: map[addrs.Provider]providers.Factory{
					mockProviderAddress: providers.FactoryFixed(mockProvider),
				},
			},
			ProviderSource: providerSource,
		}
		c := &InitCommand{
			Meta: meta,
		}

		args := []string{"-enable-pluggable-state-storage-experiment=true"}
		code := c.Run(args)
		testOutput := done(t)
		if code != 1 {
			t.Fatalf("expected code 1 exit code, got %d, output: \n%s", code, testOutput.All())
		}

		// Check output
		output := testOutput.All()
		expectedOutputs := []string{
			fmt.Sprintf("Workspace %q has not been created yet", customWorkspace),
			fmt.Sprintf("To create the custom workspace %q use the command `terraform workspace new %s`", customWorkspace, customWorkspace),
		}
		for _, expected := range expectedOutputs {
			if !strings.Contains(cleanString(output), expected) {
				t.Fatalf("expected output to include %q, but got':\n %s", expected, cleanString(output))
			}
		}

		// Assert no workspaces exist
		stateIds, err := mockProvider.MockStates.StateIds("test_store")
		if err != nil {
			t.Fatal(err)
		}
		if len(stateIds) != 0 {
			t.Fatalf("expected no workspaces, but got: %#v", stateIds)
		}

		// Assert no backend state file made due to the error
		statePath := filepath.Join(meta.DataDir(), DefaultStateFilename)
		_, err = os.Stat(statePath)
		if pathErr, ok := err.(*os.PathError); !ok || !os.IsNotExist(pathErr.Err) {
			t.Fatalf("expected backend state file to not be created, but it exists")
		}
	})

	// Test what happens when the selected workspace doesn't exist, but there are other workspaces available.
	//
	// When input is disabled (in automation, etc) Terraform cannot prompts the user to select an alternative.
	// Instead, an error is returned.
	t.Run("error when input is disabled and the selected workspace doesn't exist and other custom workspaces do exist.", func(t *testing.T) {
		// Create a temporary, uninitialized working directory with configuration including a state store
		td := t.TempDir()
		testCopyDir(t, testFixturePath("init-with-state-store"), td)
		t.Chdir(td)

		mockProvider := mockPluggableStateStorageProvider(mockSingleStateStoreSchema("test_store"))
		mockProvider.MockStates = testing_provider.NewMockStateBytesWithStateIds(
			"test_store",
			[]string{
				"foobar1",
				"foobar2",
				// Force provider to report workspaces exist
				// But default workspace doesn't exist
			},
		)

		mockProviderAddress := addrs.NewDefaultProvider("test")
		providerSource := newMockProviderSource(t, map[string][]string{
			"hashicorp/test": {"1.0.0"},
		})

		ui := testUiWrapped(t)
		view, done := testView(t)
		meta := Meta{
			Ui:                        ui,
			View:                      view,
			AllowExperimentalFeatures: true,
			testingOverrides: &testingOverrides{
				Providers: map[addrs.Provider]providers.Factory{
					mockProviderAddress: providers.FactoryFixed(mockProvider),
				},
			},
			ProviderSource: providerSource,
		}
		c := &InitCommand{
			Meta: meta,
		}

		// If input is disabled users receive an error about the missing workspace
		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
			"-input=false",
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 1 {
			t.Fatalf("expected code 1 exit code, got %d, output: \n%s", code, testOutput.All())
		}
		output := testOutput.All()
		expectedOutput := "Failed to select a workspace: Currently selected workspace \"default\" does not exist"
		if !strings.Contains(cleanString(output), expectedOutput) {
			t.Fatalf("expected output to include %q, but got':\n %s", expectedOutput, cleanString(output))
		}
		statePath := filepath.Join(meta.DataDir(), DefaultStateFilename)
		_, err := os.Stat(statePath)
		if _, ok := err.(*os.PathError); !ok {
			if err == nil {
				t.Fatalf("expected backend state file to not be created, but it exists")
			}

			t.Fatalf("unexpected error: %s", err)
		}
	})

	// Test what happens when the selected workspace doesn't exist, but there are other workspaces available.
	//
	// When input is enabled Terraform prompts the user to select an alternative.
	t.Run("if the selected workspace doesn't exist and other custom workspaces do exist, Terraform prompts the user to select a workspace .", func(t *testing.T) {
		// Create a temporary, uninitialized working directory with configuration including a state store
		td := t.TempDir()
		testCopyDir(t, testFixturePath("init-with-state-store"), td)
		t.Chdir(td)

		mockProvider := mockPluggableStateStorageProvider(mockSingleStateStoreSchema("test_store"))
		mockProvider.MockStates = testing_provider.NewMockStateBytesWithStateIds(
			"test_store",
			[]string{
				"foobar1",
				"foobar2",
				// Force provider to report workspaces exist
				// But default workspace doesn't exist
			},
		)

		mockProviderAddress := addrs.NewDefaultProvider("test")
		providerSource := newMockProviderSource(t, map[string][]string{
			"hashicorp/test": {"1.0.0"},
		})

		// Allow the test to respond to the prompt to pick an
		// existing workspace, given the selected one doesn't exist.
		uiInput, _ := testInputMap(t, map[string]string{
			"select-workspace": "1", // foobar1 in numbered list
		})

		ui := testUiWrapped(t)
		view, done := testView(t)
		meta := Meta{
			Ui:                        ui,
			View:                      view,
			AllowExperimentalFeatures: true,
			testingOverrides: &testingOverrides{
				Providers: map[addrs.Provider]providers.Factory{
					mockProviderAddress: providers.FactoryFixed(mockProvider),
				},
				UIInput: uiInput,
			},
			ProviderSource: providerSource,
		}
		c := &InitCommand{
			Meta: meta,
		}

		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 0 {
			t.Fatalf("expected code 0 exit code, got %d, output: \n%s", code, testOutput.All())
		}

		// The init command should have caused the selected workspace to change, based on the input
		// provided by the user.
		currentWorkspace, err := c.Meta.Workspace()
		if err != nil {
			t.Fatal(err)
		}
		if currentWorkspace != "foobar1" {
			t.Fatalf("expected init command to alter the selected workspace from 'default' to 'foobar1', but got: %s", currentWorkspace)
		}
	})
}

// Partial configuration allows users to override values in the config for:
// - Top level attributes (via file path or key-value pair string value for -backend-config)
// - Nested attributes (via file path value for -backend-config only, cannot be done via key-value pair)
// Reference: https://github.com/hashicorp/terraform/issues/36911
//
// It is not possible to override blocks using partial configuration, including the `provider` block
// inside the `state_store` block.
//
// The test cases below demonstrate what's possible and also what is not possible when combining
// the PSS feature with partial configuration. Only top-level attributes in the schema for the store
// implementation are overridable.
func TestInit_stateStore_newWorkingDir_partialConfiguration(t *testing.T) {
	// Reused in tests that successfully override config using -backend-config CLI flag.
	v1_2_3, _ := version.NewVersion("1.2.3")
	expectedState := &workdir.StateStoreConfigState{
		Type:               "test_store",
		ConfigRaw:          []byte("{\n      \"value\": \"OVERRIDDEN\"\n    }"), // impacted by use of -backend-config CLI flag
		Hash:               uint64(4158988729),
		ProviderSupplyMode: getproviders.ManagedByTerraform,
		Provider: &workdir.ProviderConfigState{
			Version: v1_2_3,
			Source: &tfaddr.Provider{
				Hostname:  tfaddr.DefaultProviderRegistryHost,
				Namespace: "hashicorp",
				Type:      "test",
			},
			ConfigRaw: []byte("{\n        \"region\": null\n      }"),
		},
	}

	t.Run("state store attribute - key-value pair supplied via CLI flag", func(t *testing.T) {
		// Create a temporary, uninitialized working directory with configuration including a state store
		td := t.TempDir()
		testCopyDir(t, testFixturePath("init-with-state-store"), td)
		t.Chdir(td)

		mockProvider := mockPluggableStateStorageProvider(mockSingleStateStoreSchema("test_store"))
		mockProviderAddress := addrs.NewDefaultProvider("test")

		providerSource := newMockProviderSource(t, map[string][]string{
			// The test fixture config has no version constraints, so the latest version will
			// be used; below is the 'latest' version in the test world.
			"hashicorp/test": {"1.2.3"},
		})

		ui := testUiWrapped(t)
		view, done := testView(t)
		meta := Meta{
			Ui:                        ui,
			View:                      view,
			AllowExperimentalFeatures: true,
			testingOverrides: &testingOverrides{
				Providers: map[addrs.Provider]providers.Factory{
					mockProviderAddress: providers.FactoryFixed(mockProvider),
				},
			},
			ProviderSource: providerSource,
		}
		c := &InitCommand{
			Meta: meta,
		}

		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
			`-backend-config=value=OVERRIDDEN`, // override the `value` attr in the mock hashcorp/test's test_store store
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 0 {
			t.Fatalf("expected code 0 exit code, got %d, output: \n%s", code, testOutput.All())
		}

		// Assert contents of the backend state file
		statePath := filepath.Join(meta.DataDir(), DefaultStateFilename)
		sMgr := &clistate.LocalState{Path: statePath}
		if err := sMgr.RefreshState(); err != nil {
			t.Fatal("Failed to load state:", err)
		}
		s := sMgr.State()
		if s == nil {
			t.Fatal("expected backend state file to be created, but there isn't one")
		}
		if diff := cmp.Diff(s.StateStore, expectedState); diff != "" {
			t.Fatalf("unexpected diff in backend state file's description of state store:\n%s", diff)
		}
	})

	t.Run("state store attribute - file path supplied via CLI flag", func(t *testing.T) {
		// Create a temporary, uninitialized working directory with configuration including a state store
		td := t.TempDir()
		testCopyDir(t, testFixturePath("init-with-state-store"), td)
		t.Chdir(td)

		mockProvider := mockPluggableStateStorageProvider(mockSingleStateStoreSchema("test_store"))
		mockProviderAddress := addrs.NewDefaultProvider("test")

		providerSource := newMockProviderSource(t, map[string][]string{
			// The test fixture config has no version constraints, so the latest version will
			// be used; below is the 'latest' version in the test world.
			"hashicorp/test": {"1.2.3"},
		})

		ui := testUiWrapped(t)
		view, done := testView(t)
		meta := Meta{
			Ui:                        ui,
			View:                      view,
			AllowExperimentalFeatures: true,
			testingOverrides: &testingOverrides{
				Providers: map[addrs.Provider]providers.Factory{
					mockProviderAddress: providers.FactoryFixed(mockProvider),
				},
			},
			ProviderSource: providerSource,
		}
		c := &InitCommand{
			Meta: meta,
		}

		// Create file with partial configuration
		partialConfigPath := filepath.Join(td, "overrides.test_store.tfbackend")
		cfg := `value = "OVERRIDDEN"`
		if err := os.WriteFile(partialConfigPath, []byte(cfg), 0644); err != nil {
			t.Fatalf("err: %s", err)
		}

		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
			fmt.Sprintf(`-backend-config=%s`, partialConfigPath), // override the `value` attr in the mock hashcorp/test's test_store store
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 0 {
			t.Fatalf("expected code 0 exit code, got %d, output: \n%s", code, testOutput.All())
		}

		// Assert contents of the backend state file
		statePath := filepath.Join(meta.DataDir(), DefaultStateFilename)
		sMgr := &clistate.LocalState{Path: statePath}
		if err := sMgr.RefreshState(); err != nil {
			t.Fatal("Failed to load state:", err)
		}
		s := sMgr.State()
		if s == nil {
			t.Fatal("expected backend state file to be created, but there isn't one")
		}

		if diff := cmp.Diff(s.StateStore, expectedState); diff != "" {
			t.Fatalf("unexpected diff in backend state file's description of state store:\n%s", diff)
		}
	})

	t.Run("cannot impact provider configuration - key-value pair supplied via CLI flag", func(t *testing.T) {
		// Create a temporary, uninitialized working directory with configuration including a state store
		td := t.TempDir()
		testCopyDir(t, testFixturePath("init-with-state-store"), td)
		t.Chdir(td)

		mockProvider := mockPluggableStateStorageProvider(mockSingleStateStoreSchema("test_store"))
		mockProviderAddress := addrs.NewDefaultProvider("test")

		providerSource := newMockProviderSource(t, map[string][]string{
			// The test fixture config has no version constraints, so the latest version will
			// be used; below is the 'latest' version in the test world.
			"hashicorp/test": {"1.2.3"},
		})

		ui := testUiWrapped(t)
		view, done := testView(t)
		meta := Meta{
			Ui:                        ui,
			View:                      view,
			AllowExperimentalFeatures: true,
			testingOverrides: &testingOverrides{
				Providers: map[addrs.Provider]providers.Factory{
					mockProviderAddress: providers.FactoryFixed(mockProvider),
				},
			},
			ProviderSource: providerSource,
		}
		c := &InitCommand{
			Meta: meta,
		}

		// Using CLI flags to supply key-value pairs
		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
			`-backend-config=provider.region=OVERRIDDEN`, // attempting to change the `region` attr in the nested provider block
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 1 {
			t.Fatalf("expected code 1 exit code, got %d, output: \n%s", code, testOutput.All())
		}

		if !strings.Contains(testOutput.Stderr(), "Invalid state store configuration argument") {
			t.Fatalf("expected error output to report the config argument was invalid, got:\n %s", testOutput.All())
		}
	})

	t.Run("cannot impact provider configuration - file path supplied via CLI flag", func(t *testing.T) {
		// Create a temporary, uninitialized working directory with configuration including a state store
		td := t.TempDir()
		testCopyDir(t, testFixturePath("init-with-state-store"), td)
		t.Chdir(td)

		mockProvider := mockPluggableStateStorageProvider(mockSingleStateStoreSchema("test_store"))
		mockProviderAddress := addrs.NewDefaultProvider("test")

		providerSource := newMockProviderSource(t, map[string][]string{
			// The test fixture config has no version constraints, so the latest version will
			// be used; below is the 'latest' version in the test world.
			"hashicorp/test": {"1.2.3"},
		})

		ui := testUiWrapped(t)
		view, done := testView(t)
		meta := Meta{
			Ui:                        ui,
			View:                      view,
			AllowExperimentalFeatures: true,
			testingOverrides: &testingOverrides{
				Providers: map[addrs.Provider]providers.Factory{
					mockProviderAddress: providers.FactoryFixed(mockProvider),
				},
			},
			ProviderSource: providerSource,
		}
		c := &InitCommand{
			Meta: meta,
		}

		// Create file with partial configuration
		partialConfigPath := filepath.Join(td, "overrides.test_store.tfbackend")
		cfg := `provider "test" {
  value = "OVERRIDDEN"
}`
		if err := os.WriteFile(partialConfigPath, []byte(cfg), 0644); err != nil {
			t.Fatalf("err: %s", err)
		}

		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
			fmt.Sprintf(`-backend-config=%s`, partialConfigPath), // override the `value` attr in the mock hashcorp/test's test_store store
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 1 {
			t.Fatalf("expected code 1 exit code, got %d, output: \n%s", code, testOutput.All())
		}

		expectedErrorMsgs := []string{
			// We've detected that the user is attempting to override the provider block in a state_store block.
			"Error: Cannot partially override provider configuration in a state store block.",

			// More generic error due to the state store's schema not including the provider block in it's schema.
			"Error: Unsupported block type",
		}
		for _, expected := range expectedErrorMsgs {
			if !strings.Contains(testOutput.Stderr(), expected) {
				t.Fatalf("expected error output to include %q, got:\n %s", expected, testOutput.All())
			}
		}
	})
}

// Testing init's behaviors when approving a new state store provider when a workspace is initialized for the first time.
func TestInit_stateStore_newWorkingDir_interactiveProviderApproval(t *testing.T) {
	t.Run("users do not need to approve trusting a state store provider if it's installed from local archive", func(t *testing.T) {
		// Create a temporary, uninitialized working directory with configuration including a state store
		td := t.TempDir()
		testCopyDir(t, testFixturePath("init-with-state-store"), td)
		t.Chdir(td)

		// This mock provider source makes Terraform think the provider is coming from a local archive,
		// so security checks are skipped.
		source := newMockProviderSource(t, map[string][]string{
			"hashicorp/test": {"1.2.3"},
		})

		mockProvider := mockPluggableStateStorageProvider(mockSingleStateStoreSchema("test_store"))
		mockProviderAddress := addrs.NewDefaultProvider("test")

		ui := testUiWrapped(t)
		view, done := testView(t)
		meta := Meta{
			Ui:                        ui,
			View:                      view,
			AllowExperimentalFeatures: true,
			testingOverrides: &testingOverrides{
				Providers: map[addrs.Provider]providers.Factory{
					mockProviderAddress: providers.FactoryFixed(mockProvider),
				},
			},
			ProviderSource: source,
		}
		c := &InitCommand{
			Meta: meta,
		}

		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 0 {
			t.Fatalf("expected code 0 exit code, got %d, output: \n%s", code, testOutput.All())
		}

		// Check output
		output := testOutput.All()
		expectedOutputs := []string{
			`Initializing the state store "test_store"...`,
			"Terraform has been successfully initialized!",
		}
		for _, expected := range expectedOutputs {
			if !strings.Contains(output, expected) {
				t.Fatalf("expected output to include %q, but got':\n %s", expected, output)
			}
		}

		// Assert the dependency lock file was created
		lockFile := filepath.Join(td, ".terraform.lock.hcl")
		_, err := os.Stat(lockFile)
		if os.IsNotExist(err) {
			t.Fatal("expected dependency lock file to exist, but it doesn't")
		}
	})

	t.Run("prompted to approve a state store provider downloaded via HTTP", func(t *testing.T) {
		// Create a temporary, uninitialized working directory with configuration including a state store
		td := t.TempDir()
		testCopyDir(t, testFixturePath("init-with-state-store"), td)
		t.Chdir(td)

		// Set up mock provider source that mocks out downloading hashicorp/test v1.2.3 via HTTP.
		// This stops Terraform auto-approving the provider installation.
		source := newMockProviderSourceUsingTestHttpServer(t, map[string][]string{
			"hashicorp/test": {"1.2.3"},
		})

		mockProvider := mockPluggableStateStorageProvider(mockSingleStateStoreSchema("test_store"))
		mockProviderAddress := addrs.NewDefaultProvider("test")

		// Allow the test to respond to the pause in provider installation for
		// checking the state storage provider.
		uiInput, inputWriter := testInputMap(t, map[string]string{
			"approve-provider-test-1.2.3": "yes",
		})

		ui := testUiWrapped(t)
		view, done := testView(t)
		meta := Meta{
			Ui:                        ui,
			View:                      view,
			AllowExperimentalFeatures: true,
			testingOverrides: &testingOverrides{
				Providers: map[addrs.Provider]providers.Factory{
					mockProviderAddress: providers.FactoryFixed(mockProvider),
				},
				UIInput: uiInput,
			},
			ProviderSource: source,
		}
		c := &InitCommand{
			Meta: meta,
		}

		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 0 {
			t.Fatalf("expected code 0 exit code, got %d, output: \n%s", code, testOutput.All())
		}

		// Check output via view
		output := testOutput.All()
		expectedOutputs := []string{
			`Initializing the state store "test_store"...`,
			"The state store provider was approved",
			"Terraform has been successfully initialized!",
		}
		for _, expected := range expectedOutputs {
			if !strings.Contains(output, expected) {
				t.Fatalf("expected output to include %q, but got':\n %s", expected, output)
			}
		}
		// Check output when prompting for approval
		expectedInputPromptMsg := []string{
			"Do you want to use provider \"test\" (registry.terraform.io/hashicorp/test), version 1.2.3, for managing state?",
			getproviders.CurrentPlatform.String(),
			"Authentication: verified checksum",
			"h1:wlbEC2mChQZ2hhgUhl6SeVLPP7fMqOFUZAQhQ9GIIno=",
		}
		for _, expected := range expectedInputPromptMsg {
			if !strings.Contains(inputWriter.String(), expected) {
				t.Fatalf("expected the input prompt to include %q, but got':\n %s", expected, inputWriter.String())
			}
		}

		// Assert the dependency lock file was created
		lockFile := filepath.Join(td, ".terraform.lock.hcl")
		_, err := os.Stat(lockFile)
		if os.IsNotExist(err) {
			t.Fatal("expected dependency lock file to exist, but it doesn't")
		}
	})

	t.Run("not prompted to approve a state store provider downloaded via HTTP if the provider is in the dependency lock file.", func(t *testing.T) {
		// This scenario is distinct from using the -state-store-provider-override flag, which is specific to non-interactive/automation mode.
		// Here, we assert that users aren't prompted for approval if the provider hasn't been downloaded yet but it's described in the working directory's lock file.
		// This will happen when a user git clones a Terraform project from version control, or a lock file is copied from elsewhere when starting a project.

		td := t.TempDir()
		testCopyDir(t, testFixturePath("state-store-unchanged/provider-managed-by-terraform"), td)
		t.Chdir(td)

		// The dependency lock file already exists.
		lockFile := filepath.Join(td, depsfile.LockFilePath)
		_, err := os.Stat(lockFile)
		if os.IsNotExist(err) {
			t.Fatal("expected dependency lock file to exist, but it doesn't")
		}

		// Set up mock provider source that mocks out downloading hashicorp/test v1.2.3 via HTTP.
		// This stops Terraform auto-approving the provider installation.
		source := newMockProviderSourceUsingTestHttpServer(t, map[string][]string{
			"hashicorp/test": {"1.2.3"},
		})

		mockProvider := mockPluggableStateStorageProvider(mockSingleStateStoreSchema("test_store"))
		mockProviderAddress := addrs.NewDefaultProvider("test")

		ui := testUiWrapped(t)
		view, done := testView(t)
		meta := Meta{
			Ui:                        ui,
			View:                      view,
			AllowExperimentalFeatures: true,
			testingOverrides: &testingOverrides{
				Providers: map[addrs.Provider]providers.Factory{
					mockProviderAddress: providers.FactoryFixed(mockProvider),
				},
			},
			ProviderSource: source,
		}
		c := &InitCommand{
			Meta: meta,
		}

		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 0 {
			t.Fatalf("expected code 0 exit code, got %d, output: \n%s", code, testOutput.All())
		}

		// Check output via view
		output := testOutput.All()
		expectedOutputs := []string{
			`Initializing the state store "test_store"...`,
			"Terraform has been successfully initialized!",
		}
		for _, expected := range expectedOutputs {
			if !strings.Contains(output, expected) {
				t.Fatalf("expected output to include %q, but got':\n %s", expected, output)
			}
		}
	})

	t.Run("approval prompt reports provider as unauthorized if no hashes returned from the HTTP mirror", func(t *testing.T) {
		// Create a temporary, uninitialized working directory with configuration including a state store
		td := t.TempDir()
		testCopyDir(t, testFixturePath("init-with-state-store"), td)
		t.Chdir(td)

		// The network mirror the provider will be downloaded from will not return any hashes, so
		// Terraform won't have any way to check the provider's authenticity.
		// This affects the prompt for approval, which this test case focuses on.
		returnApprovedHashes := false
		source := newHTTPMirrorProviderSourceUsingTestHttpServer(t, map[string][]string{
			"hashicorp/test": {"1.2.3"},
		}, returnApprovedHashes)

		mockProvider := mockPluggableStateStorageProvider(mockSingleStateStoreSchema("test_store"))
		mockProviderAddress := addrs.NewDefaultProvider("test")

		// Allow the test to respond to the pause in provider installation for
		// checking the state storage provider.
		uiInput, inputWriter := testInputMap(t, map[string]string{
			"approve-provider-test-1.2.3": "yes",
		})

		ui := testUiWrapped(t)
		view, done := testView(t)
		meta := Meta{
			Ui:                        ui,
			View:                      view,
			AllowExperimentalFeatures: true,
			testingOverrides: &testingOverrides{
				Providers: map[addrs.Provider]providers.Factory{
					mockProviderAddress: providers.FactoryFixed(mockProvider),
				},
				UIInput: uiInput,
			},
			ProviderSource: source,
		}
		c := &InitCommand{
			Meta: meta,
		}

		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 0 {
			t.Fatalf("expected code 0 exit code, got %d, output: \n%s", code, testOutput.All())
		}

		// Check output via view
		output := testOutput.All()
		expectedOutputs := []string{
			`Initializing the state store "test_store"...`,
			"The state store provider was approved",
			"Terraform has been successfully initialized!",
		}
		for _, expected := range expectedOutputs {
			if !strings.Contains(output, expected) {
				t.Fatalf("expected output to include %q, but got':\n %s", expected, output)
			}
		}
		// Check output when prompting for approval
		expectedInputPromptMsg := []string{
			"Do you want to use provider \"test\" (registry.terraform.io/hashicorp/test), version 1.2.3, for managing state?",
			getproviders.CurrentPlatform.String(),
			"Authentication: unauthenticated",
			"h1:wlbEC2mChQZ2hhgUhl6SeVLPP7fMqOFUZAQhQ9GIIno=",
		}
		for _, expected := range expectedInputPromptMsg {
			if !strings.Contains(inputWriter.String(), expected) {
				t.Fatalf("expected the input prompt to include %q, but got':\n %s", expected, inputWriter.String())
			}
		}

		// Assert the dependency lock file was created
		lockFile := filepath.Join(td, ".terraform.lock.hcl")
		_, err := os.Stat(lockFile)
		if os.IsNotExist(err) {
			t.Fatal("expected dependency lock file to exist, but it doesn't")
		}
	})

	t.Run("users can reject a state store provider downloaded via HTTP", func(t *testing.T) {
		// Create a temporary, uninitialized working directory with configuration including a state store
		td := t.TempDir()
		testCopyDir(t, testFixturePath("init-with-state-store"), td)
		t.Chdir(td)

		// Set up mock provider source that mocks out downloading hashicorp/test v1.2.3 via HTTP.
		// This stops Terraform auto-approving the provider installation.
		source := newMockProviderSourceUsingTestHttpServer(t, map[string][]string{
			"hashicorp/test": {"1.2.3"},
		})

		mockProvider := mockPluggableStateStorageProvider(mockSingleStateStoreSchema("test_store"))
		mockProviderAddress := addrs.NewDefaultProvider("test")

		// Allow the test to respond to the pause in provider installation for
		// checking the state storage provider.
		uiInput, inputWriter := testInputMap(t, map[string]string{
			"approve-provider-test-1.2.3": "no",
		})

		ui := testUiWrapped(t)
		view, done := testView(t)
		meta := Meta{
			Ui:                        ui,
			View:                      view,
			AllowExperimentalFeatures: true,
			testingOverrides: &testingOverrides{
				Providers: map[addrs.Provider]providers.Factory{
					mockProviderAddress: providers.FactoryFixed(mockProvider),
				},
				UIInput: uiInput,
			},
			ProviderSource: source,
		}
		c := &InitCommand{
			Meta: meta,
		}

		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 1 {
			t.Fatalf("expected code 1 exit code, got %d, output: \n%s", code, testOutput.All())
		}

		// Check output via view
		output := testOutput.All()
		expectedOutputs := []string{
			"The state store provider was rejected",
			`Error: State store provider "test" (registry.terraform.io/hashicorp/test) was not approved, so init cannot continue.`,
		}
		for _, expected := range expectedOutputs {
			if !strings.Contains(output, expected) {
				t.Fatalf("expected output to include %q, but got':\n %s", expected, output)
			}
		}
		// Check output when prompting for approval
		expectedInputPromptMsg := []string{
			"Do you want to use provider \"test\" (registry.terraform.io/hashicorp/test), version 1.2.3, for managing state?",
			getproviders.CurrentPlatform.String(),
			"Authentication: verified checksum",
			"h1:wlbEC2mChQZ2hhgUhl6SeVLPP7fMqOFUZAQhQ9GIIno=",
		}
		for _, expected := range expectedInputPromptMsg {
			if !strings.Contains(inputWriter.String(), expected) {
				t.Fatalf("expected the input prompt to include %q, but got':\n %s", expected, inputWriter.String())
			}
		}

		// Assert the dependency lock file was not created
		lockFile := filepath.Join(td, ".terraform.lock.hcl")
		_, err := os.Stat(lockFile)
		if !os.IsNotExist(err) {
			t.Fatal("expected dependency lock file to not exist, but it does")
		}
	})

	t.Run("re-prompt to approve a provider after rejecting that provider in a previous init", func(t *testing.T) {
		// Create a temporary, uninitialized working directory with configuration including a state store
		td := t.TempDir()
		testCopyDir(t, testFixturePath("init-with-state-store"), td)
		t.Chdir(td)

		// Set up mock provider source that mocks out downloading hashicorp/test v1.2.3 via HTTP.
		// This stops Terraform auto-approving the provider installation.
		mockProviderAddress := addrs.NewDefaultProvider("test")
		mockProviderVersion := getproviders.MustParseVersion("1.2.3")
		source := newMockProviderSourceUsingTestHttpServer(t, map[string][]string{
			"hashicorp/test": {"1.2.3"},
		})

		// Set up providers for use in the second init attempt.
		mockProvider := mockPluggableStateStorageProvider(mockSingleStateStoreSchema("test_store"))

		ui := testUiWrapped(t)
		view, done := testView(t)
		meta := Meta{
			Ui:                        ui,
			View:                      view,
			AllowExperimentalFeatures: true,
			testingOverrides: &testingOverrides{
				Providers: map[addrs.Provider]providers.Factory{
					mockProviderAddress: providers.FactoryFixed(mockProvider),
				},
			},
			ProviderSource: source,
		}
		c := &InitCommand{
			Meta: meta,
		}

		// Init number 1 - reject the provider
		uiInput, _ := testInputMap(t, map[string]string{
			"approve-provider-test-1.2.3": "no",
		})
		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
		}
		c.Meta.testingOverrides.UIInput = uiInput
		code := c.Run(args)
		testOutput := done(t)
		if code != 1 {
			t.Fatalf("expected code 1 exit code, got %d, output: \n%s", code, testOutput.All())
		}
		output := testOutput.All()
		expectedOutputs := []string{
			"The state store provider was rejected",
			`Error: State store provider "test" (registry.terraform.io/hashicorp/test) was not approved, so init cannot continue.`,
		}
		for _, expectedOutput := range expectedOutputs {
			if !strings.Contains(output, expectedOutput) {
				t.Fatalf("expected output to include %q, but got':\n %s", expectedOutput, output)
			}
		}

		// The rejected provider is present in the local cache after being rejected.
		// However, this doesn't stop the user being prompted again.
		cacheDir := meta.providerLocalCacheDir()
		gotPackages := cacheDir.AllAvailablePackages()
		wantPackages := map[addrs.Provider][]providercache.CachedProvider{
			// "between" wasn't previously installed at all, so we installed
			// the newest available version that matched the version constraints.
			mockProviderAddress: {
				{
					Provider:   mockProviderAddress,
					Version:    mockProviderVersion,
					PackageDir: expectedPackageInstallPath(mockProviderAddress.Type, mockProviderVersion.String(), false),
				},
			},
		}
		if diff := cmp.Diff(wantPackages, gotPackages); diff != "" {
			t.Errorf("wrong cache directory contents after upgrade\n%s", diff)
		}

		// Init number 2 - re-prompted for approval
		uiInput, _ = testInputMap(t, map[string]string{
			"approve-provider-test-1.2.3": "yes",
		})
		args = []string{
			"-enable-pluggable-state-storage-experiment=true",
		}
		ui = testUiWrapped(t)
		view, done = testView(t)
		c.Meta.testingOverrides.UIInput = uiInput
		c.Ui = ui
		c.View = view
		code = c.Run(args)
		testOutput = done(t)
		if code != 0 {
			t.Fatalf("expected code 0 exit code, got %d, output: \n%s", code, testOutput.All())
		}
		output = testOutput.All()
		expectedOutputs = []string{
			`Initializing the state store "test_store"...`,
			"The state store provider was approved",
			"Terraform has been successfully initialized!",
		}
		for _, expected := range expectedOutputs {
			if !strings.Contains(output, expected) {
				t.Fatalf("expected output to include %q, but got':\n %s", expected, output)
			}
		}
	})
}

// Test what happens when a child module also uses the provider for state storage and has a version constraint
// that doesn't match the version constraint in the root module. We need the state store provider to be installed before
// child modules are downloaded, so in this scenario we expect an error to happen.
func TestInit_stateStore_versionConstraintChildModule(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-with-state-store-in-child-module"), td)
	t.Chdir(td)

	source := newMockProviderSource(t, map[string][]string{
		"hashicorp/test": {
			"1.0.0", // Satisfies constraint in root module only
			"2.0.0", // Satisfies constraint in child module only
		},
	})

	mockProvider := mockPluggableStateStorageProvider(mockSingleStateStoreSchema("test_store"))
	mockProviderAddress := addrs.NewDefaultProvider("test")

	ui := testUiWrapped(t)
	view, done := testView(t)
	meta := Meta{
		Ui:                        ui,
		View:                      view,
		AllowExperimentalFeatures: true,
		testingOverrides: &testingOverrides{
			Providers: map[addrs.Provider]providers.Factory{
				mockProviderAddress: providers.FactoryFixed(mockProvider),
			},
		},
		ProviderSource: source,
	}
	c := &InitCommand{
		Meta: meta,
	}

	args := []string{
		"-enable-pluggable-state-storage-experiment",
	}
	code := c.Run(args)
	testOutput := done(t)
	if code != 1 {
		t.Fatalf("expected code 1 exit code, got %d, output: \n%s", code, testOutput.All())
	}

	// Check output- this must be done separately for stdout and stderr due to
	// the interleaving of output caused in tests by (TestOutput).All()

	// Check stdout
	stdout := testOutput.Stdout()
	expectedOutput := `Installing provider hashicorp/test (< 2.0.0) for state store "test_store"...
- Finding hashicorp/test versions matching "< 2.0.0"...
- Installing hashicorp/test v1.0.0...
- Installed hashicorp/test v1.0.0 (verified checksum)

Initializing the state store "test_store"...

Initializing modules...
- child in child

Initializing provider plugins...
- Reusing version 1.0.0 of hashicorp/test from the dependency lock file
`

	if stdout != expectedOutput {
		t.Errorf("didn't get expected output")
		diff := cmp.Diff(expectedOutput, stdout)
		t.Fatalf("unexpected diff in output:\n%s", diff)
	}

	// Check stderr
	stderr := testOutput.Stderr()
	expectedError := `
Error: Unable to download the provider used for state storage

Provider "test" (hashicorp/test) is used to store state, so the root module's
version constraints take precedence when downloading the provider. Terraform
encountered an error that suggests that version constraint may be conflicting
with a version constraint from a child module. If you want to upgrade the
provider used for state storage you must use the following command:
    terraform state migrate -upgrade

Error from the installer: locked provider
registry.terraform.io/hashicorp/test 1.0.0 does not match configured version
constraint >= 2.0.0, < 2.0.0; must use terraform init -upgrade to allow
selection of new versions

To see which modules are currently depending on hashicorp/test and what
versions are specified, run the following command:
    terraform providers
`

	if stderr != expectedError {
		t.Errorf("didn't get expected error output")
		diff := cmp.Diff(expectedError, stderr)
		t.Fatalf("unexpected diff in error output:\n%s", diff)
	}
}

// Testing init's behaviors when, in automation, we're approving a new state store provider when a workspace is initialized for the first time.
func TestInit_stateStore_newWorkingDir_inAutomationProviderApproval(t *testing.T) {
	t.Run("users do not need to approve trusting a state store provider if it's installed from local archive", func(t *testing.T) {
		// Create a temporary, uninitialized working directory with configuration including a state store
		td := t.TempDir()
		testCopyDir(t, testFixturePath("init-with-state-store"), td)
		t.Chdir(td)

		// This mock provider source makes Terraform think the provider is coming from a local archive,
		// so security checks are skipped.
		source := newMockProviderSource(t, map[string][]string{
			"hashicorp/test": {"1.2.3"},
		})

		mockProvider := mockPluggableStateStorageProvider(mockSingleStateStoreSchema("test_store"))
		mockProviderAddress := addrs.NewDefaultProvider("test")

		ui := testUiWrapped(t)
		view, done := testView(t)
		meta := Meta{
			Ui:                        ui,
			View:                      view,
			AllowExperimentalFeatures: true,
			testingOverrides: &testingOverrides{
				Providers: map[addrs.Provider]providers.Factory{
					mockProviderAddress: providers.FactoryFixed(mockProvider),
				},
			},
			ProviderSource: source,
		}
		c := &InitCommand{
			Meta: meta,
		}

		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
			"-input=false", // Simulate running in automation where input is disabled
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 0 {
			t.Fatalf("expected code 0 exit code, got %d, output: \n%s", code, testOutput.All())
		}

		// Check output
		output := testOutput.All()
		expectedOutputs := []string{
			`Initializing the state store "test_store"...`,
			"Terraform has been successfully initialized!",
		}
		for _, expected := range expectedOutputs {
			if !strings.Contains(output, expected) {
				t.Fatalf("expected output to include %q, but got':\n %s", expected, output)
			}
		}

		// Assert the dependency lock file was created
		lockFile := filepath.Join(td, ".terraform.lock.hcl")
		_, err := os.Stat(lockFile)
		if os.IsNotExist(err) {
			t.Fatal("expected dependency lock file to exist, but it doesn't")
		}
	})

	t.Run("users approve trusting a state store provider downloaded via HTTP by supplying locks via -state-provider-lock flag", func(t *testing.T) {
		// Create a temporary, uninitialized working directory with configuration including a state store
		td := t.TempDir()
		testCopyDir(t, testFixturePath("init-with-state-store"), td)
		t.Chdir(td)

		// Set up mock provider source that mocks out hashicorp/test via HTTP.
		// This stops Terraform auto-approving the provider installation.
		mockProviderAddress := addrs.NewDefaultProvider("test")
		expectedVersion := "1.2.3"
		source := newMockProviderSourceUsingTestHttpServer(t, map[string][]string{
			"hashicorp/test": {expectedVersion, "9.9.9"}, // Extra version - expected version is downloaded, not the latest
		})
		mockProvider := mockPluggableStateStorageProvider(mockSingleStateStoreSchema("test_store"))

		ui := testUiWrapped(t)
		view, done := testView(t)
		meta := Meta{
			Ui:                        ui,
			View:                      view,
			AllowExperimentalFeatures: true,
			testingOverrides: &testingOverrides{
				Providers: map[addrs.Provider]providers.Factory{
					mockProviderAddress: providers.FactoryFixed(mockProvider),
				},
			},
			ProviderSource: source,
		}
		c := &InitCommand{
			Meta: meta,
		}

		// Create supplemental lock file to be used with the -state-provider-lock flag
		// To avoid this being confused with the lock file in the working directory,
		// this is made in a second temp directory away from other files in this test.
		td2 := t.TempDir()
		lockFileName := filepath.Join(td2, ".terraform.lock.hcl")
		suppliedLockFileVersion := getproviders.MustParseVersion(expectedVersion)
		locks := depsfile.NewLocks()
		locks.SetProvider(
			mockProviderAddress,
			suppliedLockFileVersion,
			getproviders.MustParseVersionConstraints("> 1.0.0"),
			[]getproviders.Hash{
				getproviders.HashScheme1.New("wlbEC2mChQZ2hhgUhl6SeVLPP7fMqOFUZAQhQ9GIIno="),
			},
		)
		depsfile.SaveLocksToFile(locks, lockFileName)

		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
			"-input=false", // Simulate running in automation where input is disabled
			fmt.Sprintf("-state-provider-lock-file=%s", lockFileName),
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 0 {
			t.Fatalf("expected code 0 exit code, got %d, output: \n%s", code, testOutput.All())
		}

		// Check output via view
		output := testOutput.All()
		expectedOutputs := []string{
			`Initializing the state store "test_store"...`,
			"The state store provider was approved automatically",
			"Terraform has been successfully initialized!",
		}
		for _, expected := range expectedOutputs {
			if !strings.Contains(output, expected) {
				t.Fatalf("expected output to include %q, but got':\n %s", expected, output)
			}
		}

		// Assert the dependency lock file was created
		// and it contains the state store provider version described by the -state-provider-lock-file flag
		lockFile := filepath.Join(td, ".terraform.lock.hcl")
		locks, lockDiags := depsfile.LoadLocksFromFile(lockFile)
		if lockDiags.HasErrors() {
			t.Fatal("expected dependency lock file to exist, but got errors loading it:", lockDiags.Err())
		}
		gotLock := locks.Provider(mockProviderAddress)
		if gotLock == nil {
			t.Fatalf("expected dependency lock file to contain the state store provider %s, but it doesn't", mockProviderAddress.ForDisplay())
		}
		if !gotLock.Version().Same(suppliedLockFileVersion) {
			t.Fatalf("expected dependency lock file to contain version %s for provider %s that was supplied via the -state-provider-lock-file flag, but got version %s", suppliedLockFileVersion, mockProviderAddress.ForDisplay(), gotLock.Version())
		}
	})

	t.Run("a state store provider downloaded via HTTP can be automatically approved if it already exists in the .terraform.lock.hcl file", func(t *testing.T) {
		// Create a temporary, uninitialized working directory with configuration including a state store
		td := t.TempDir()
		testCopyDir(t, testFixturePath("init-with-state-store"), td)
		t.Chdir(td)

		// Set up mock provider source that mocks out hashicorp/test via HTTP.
		// This stops Terraform auto-approving the provider installation.
		mockProviderAddress := addrs.NewDefaultProvider("test")
		expectedVersion := "1.2.3"
		source := newMockProviderSourceUsingTestHttpServer(t, map[string][]string{
			"hashicorp/test": {expectedVersion, "9.9.9"}, // Extra version - expected version is downloaded, not the latest
		})
		mockProvider := mockPluggableStateStorageProvider(mockSingleStateStoreSchema("test_store"))

		ui := testUiWrapped(t)
		view, done := testView(t)
		meta := Meta{
			Ui:                        ui,
			View:                      view,
			AllowExperimentalFeatures: true,
			testingOverrides: &testingOverrides{
				Providers: map[addrs.Provider]providers.Factory{
					mockProviderAddress: providers.FactoryFixed(mockProvider),
				},
			},
			ProviderSource: source,
		}
		c := &InitCommand{
			Meta: meta,
		}

		// Create a local .terraform.lock.hcl file that already contains the state store provider version
		lockFileName := ".terraform.lock.hcl"
		suppliedLockFileVersion := getproviders.MustParseVersion(expectedVersion)
		locks := depsfile.NewLocks()
		locks.SetProvider(
			mockProviderAddress,
			suppliedLockFileVersion,
			getproviders.MustParseVersionConstraints("> 1.0.0"),
			[]getproviders.Hash{
				getproviders.HashScheme1.New("wlbEC2mChQZ2hhgUhl6SeVLPP7fMqOFUZAQhQ9GIIno="),
			},
		)
		depsfile.SaveLocksToFile(locks, lockFileName)

		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
			"-input=false", // Simulate running in automation where input is disabled
			// -state-provider-lock-file flag isn't used; this test shows it can fall back to using the .terraform.lock.hcl file in the working directory
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 0 {
			t.Fatalf("expected code 0 exit code, got %d, output: \n%s", code, testOutput.All())
		}

		// Check output via view
		output := testOutput.All()
		expectedOutputs := []string{
			`Initializing the state store "test_store"...`,
			"Terraform has been successfully initialized!",
		}
		for _, expected := range expectedOutputs {
			if !strings.Contains(output, expected) {
				t.Fatalf("expected output to include %q, but got':\n %s", expected, output)
			}
		}
		// No need for assertions about the dependency lock file
		// as it was created during test setup.
	})

	t.Run("error if the lock file supplied by the -state-provider-lock-file flag doesn't contain the state store provider", func(t *testing.T) {
		// Create a temporary, uninitialized working directory with configuration including a state store
		td := t.TempDir()
		testCopyDir(t, testFixturePath("init-with-state-store"), td)
		t.Chdir(td)

		// Set up mock provider source that mocks out hashicorp/test via HTTP.
		// This stops Terraform auto-approving the provider installation.
		mockProviderAddress := addrs.NewDefaultProvider("test")
		expectedVersion := "1.2.3"
		source := newMockProviderSourceUsingTestHttpServer(t, map[string][]string{
			"hashicorp/test": {expectedVersion},
		})
		mockProvider := mockPluggableStateStorageProvider(mockSingleStateStoreSchema("test_store"))

		ui := testUiWrapped(t)
		view, done := testView(t)
		meta := Meta{
			Ui:                        ui,
			View:                      view,
			AllowExperimentalFeatures: true,
			testingOverrides: &testingOverrides{
				Providers: map[addrs.Provider]providers.Factory{
					mockProviderAddress: providers.FactoryFixed(mockProvider),
				},
			},
			ProviderSource: source,
		}
		c := &InitCommand{
			Meta: meta,
		}

		// Create supplemental lock file to be used with the -state-provider-lock flag
		// To avoid this being confused with the lock file in the working directory,
		// this is made in a second temp directory away from other files in this test.
		td2 := t.TempDir()
		lockFileName := filepath.Join(td2, ".terraform.lock.hcl")

		// It DOESN'T contain the state store provider hashicorp/test though, causing an error.
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
			"-enable-pluggable-state-storage-experiment=true",
			"-input=false", // Simulate running in automation where input is disabled
			fmt.Sprintf("-state-provider-lock-file=%s", lockFileName),
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 1 {
			t.Fatalf("expected code 1 exit code, got %d, output: \n%s", code, testOutput.All())
		}

		// Check output via view
		output := cleanString(testOutput.All())
		expectedOutputs := []string{
			"Error: State store provider not described in dependency lock file supplied via -state-provider-lock-file flag",
			fmt.Sprintf(`Terraform checked the lock file at %q, supplied via the -state-provider-lock-file flag, but could not find the state store provider "test" (hashicorp/test).`, lockFileName),
		}
		for _, expected := range expectedOutputs {
			if !strings.Contains(output, expected) {
				t.Fatalf("expected output to include %q, but got':\n %s", expected, output)
			}
		}
	})

	t.Run("error if the state store lock is supplied by neither a pre-existing lock nor the -state-provider-lock-file flag", func(t *testing.T) {
		// Create a temporary, uninitialized working directory with configuration including a state store
		td := t.TempDir()
		testCopyDir(t, testFixturePath("init-with-state-store"), td)
		t.Chdir(td)

		// Set up mock provider source that mocks out hashicorp/test via HTTP.
		// This stops Terraform auto-approving the provider installation.
		mockProviderAddress := addrs.NewDefaultProvider("test")
		expectedVersion := "1.2.3"
		source := newMockProviderSourceUsingTestHttpServer(t, map[string][]string{
			"hashicorp/test": {expectedVersion},
		})
		mockProvider := mockPluggableStateStorageProvider(mockSingleStateStoreSchema("test_store"))

		ui := testUiWrapped(t)
		view, done := testView(t)
		meta := Meta{
			Ui:                        ui,
			View:                      view,
			AllowExperimentalFeatures: true,
			testingOverrides: &testingOverrides{
				Providers: map[addrs.Provider]providers.Factory{
					mockProviderAddress: providers.FactoryFixed(mockProvider),
				},
			},
			ProviderSource: source,
		}
		c := &InitCommand{
			Meta: meta,
		}

		// Confirm the .terraform.lock.hcl file doesn't exist before the test runs
		// Therefore this isn't an adequate fallback source of locks for the state store provider, causing an error.
		_, err := os.Stat(filepath.Join(td, ".terraform.lock.hcl"))
		if !os.IsNotExist(err) {
			t.Fatal("expected .terraform.lock.hcl file to not exist, but got an unrelated error:", err)
		}

		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
			"-input=false", // Simulate running in automation where input is disabled
			//-state-provider-lock-file is not used, and there's no .terraform.lock.hcl file, so no locks are supplied for the state store provider
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
			`Terraform used the working directory's lock file by default, but it was empty or did not exist.`,
			`When performing a "terraform init" command in automation, make sure to supply a lock file for the state store provider using the -state-provider-lock-file flag.`,
		}
		for _, expected := range expectedOutputs {
			if !strings.Contains(output, expected) {
				t.Fatalf("expected output to include %q, but got':\n %s", expected, output)
			}
		}
	})
}

// Test a scenario where a state store is introduced via `init -reconfigure` in a working directory
// that previously used the local backend and contains local state files. These are a special case
// that's handled outside of the usual state migration UX.
func TestInit_stateStore_reconfigureLeadingToMigrationOfLocalState(t *testing.T) {
	// Create a temporary, uninitialized working directory with configuration including a state store
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-with-state-store"), td)
	t.Chdir(td)

	// Make a pre-existing local state file
	statePath := filepath.Join(td, DefaultStateFilename)
	preExistingState := `{
    "version": 4,
    "terraform_version": "1.16.0",
    "serial": 2,
    "lineage": "43f9a61a-43ae-7158-0a90-87fc8a4ab5e9",
    "outputs": {
        "greeting": {
            "value": "hello world",
            "type": "string"
        }
    },
    "resources": [],
    "check_results": null
}`
	if err := os.WriteFile(statePath, []byte(preExistingState), 0644); err != nil {
		t.Fatalf("failed to write pre-existing state file: %s", err)
	}

	// Make a pre-existing backend state file describing using the local backend
	if err := os.Mkdir(filepath.Join(td, DefaultDataDir), 0755); err != nil {
		t.Fatal(err)
	}
	backendStatePath := filepath.Join(td, DefaultDataDir, DefaultStateFilename)
	backendState := `{
  "version": 3,
  "terraform_version": "1.14.0",
  "backend": {
    "type": "local",
    "config": {
      "path": null,
      "workspace_dir": "my-other-workspaces"
    },
    "hash": 549668327
  }
}`
	if err := os.WriteFile(backendStatePath, []byte(backendState), 0644); err != nil {
		t.Fatalf("failed to write pre-existing backend state file: %s", err)
	}

	// This mock provider source makes Terraform think the provider is coming from a local archive,
	// so security checks are skipped.
	source := newMockProviderSource(t, map[string][]string{
		"hashicorp/test": {"1.2.3"},
	})

	mockProvider := mockPluggableStateStorageProvider(mockSingleStateStoreSchema("test_store"))
	mockProvider.MockStates = testing_provider.NewMockStateBytesWithStateIds("test_store", []string{"default"})
	mockProviderAddress := addrs.NewDefaultProvider("test")

	// Allow the test to respond to the pause in provider installation for
	// checking the state storage provider.
	uiInput, _ := testInputMap(t, map[string]string{
		"backend-migrate-copy-to-empty": "yes",
	})

	ui := testUiWrapped(t)
	view, done := testView(t)
	meta := Meta{
		Ui:                        ui,
		View:                      view,
		AllowExperimentalFeatures: true,
		testingOverrides: &testingOverrides{
			Providers: map[addrs.Provider]providers.Factory{
				mockProviderAddress: providers.FactoryFixed(mockProvider),
			},
			UIInput: uiInput,
		},
		ProviderSource: source,
	}
	c := &InitCommand{
		Meta: meta,
	}

	args := []string{
		"-reconfigure",
		"-enable-pluggable-state-storage-experiment",
	}

	code := c.Run(args)
	testOutput := done(t)
	if code != 0 {
		t.Fatalf("expected code 0 exit code, got %d, output: \n%s", code, testOutput.All())
	}

	// Check output
	output := testOutput.All()
	expectedOutputs := []string{
		`Initializing the state store "test_store"...`,
		"Terraform has been successfully initialized!",
	}
	for _, expected := range expectedOutputs {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected output to include %q, but got':\n %s", expected, output)
		}
	}

	// Assert the dependency lock file was created
	lockFile := filepath.Join(td, ".terraform.lock.hcl")
	_, err := os.Stat(lockFile)
	if os.IsNotExist(err) {
		t.Fatal("expected dependency lock file to exist, but it doesn't")
	}
}

// Testing init's behaviors with `state_store` when run in a working directory where
// nothing has changed since the last init.
func TestInit_stateStore_configUnchanged(t *testing.T) {
	// This matches the backend state test fixture in "state-store-unchanged/provider-managed-by-terraform"
	v1_2_3, _ := version.NewVersion("1.2.3")

	t.Run("successful init - no changes detected", func(t *testing.T) {
		// Create a temporary working directory with state store configuration
		// that matches the backend state file
		td := t.TempDir()
		testCopyDir(t, testFixturePath("state-store-unchanged/provider-managed-by-terraform"), td)
		t.Chdir(td)

		mockProvider := mockPluggableStateStorageProvider(mockSingleStateStoreSchema("test_store"))
		mockProvider.MockStates = testing_provider.NewMockStateBytesWithStateIds("test_store", []string{"default"})
		// If the working directory was previously initialized successfully then at least
		// one workspace is guaranteed to exist when a user is re-running init with no config
		// changes since last init. So this test says `default` exists.
		mockProvider.GetStatesResponse = &providers.GetStatesResponse{
			States: []string{"default"},
		}
		mockProviderAddress := addrs.NewDefaultProvider("test")
		providerSource := newMockProviderSource(t, map[string][]string{
			"hashicorp/test": {"1.2.3"}, // Matches provider version in backend state file fixture
		})

		ui := testUiWrapped(t)
		view, done := testView(t)
		meta := Meta{
			Ui:                        ui,
			View:                      view,
			AllowExperimentalFeatures: true,
			testingOverrides: &testingOverrides{
				Providers: map[addrs.Provider]providers.Factory{
					mockProviderAddress: providers.FactoryFixed(mockProvider),
				},
			},
			ProviderSource: providerSource,
		}
		c := &InitCommand{
			Meta: meta,
		}

		// Before running init, confirm the contents of the backend state file before
		statePath := filepath.Join(meta.DataDir(), DefaultStateFilename)
		sMgr := &clistate.LocalState{Path: statePath}
		if err := sMgr.RefreshState(); err != nil {
			t.Fatal("Failed to load state:", err)
		}
		s := sMgr.State()
		if s == nil {
			t.Fatal("expected backend state file to be present, but there isn't one")
		}
		expectedState := &workdir.StateStoreConfigState{
			Type:               "test_store",
			ConfigRaw:          []byte("{\n            \"value\": \"foobar\"\n        }"),
			Hash:               uint64(4158988729),
			ProviderSupplyMode: getproviders.ManagedByTerraform,
			Provider: &workdir.ProviderConfigState{
				Version: v1_2_3,
				Source: &tfaddr.Provider{
					Hostname:  tfaddr.DefaultProviderRegistryHost,
					Namespace: "hashicorp",
					Type:      "test",
				},
				ConfigRaw: []byte("{\n                \"region\": null\n            }"),
			},
		}
		if diff := cmp.Diff(s.StateStore, expectedState); diff != "" {
			t.Fatalf("unexpected diff in backend state file's description of state store:\n%s", diff)
		}

		// Run init command
		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 0 {
			t.Fatalf("expected code 0 exit code, got %d, output: \n%s", code, testOutput.All())
		}

		// Check output
		output := testOutput.All()
		expectedOutputs := []string{
			`Initializing the state store "test_store"...`,
			"Terraform has been successfully initialized!",
		}
		for _, expected := range expectedOutputs {
			if !strings.Contains(output, expected) {
				t.Fatalf("expected output to include %q, but got':\n %s", expected, output)
			}
		}

		// Confirm init was a no-op and backend state is unchanged afterwards
		if err := sMgr.RefreshState(); err != nil {
			t.Fatal("Failed to load state:", err)
		}
		s = sMgr.State()
		if diff := cmp.Diff(s.StateStore, expectedState); diff != "" {
			t.Fatalf("unexpected diff in backend state file's description of state store:\n%s", diff)
		}
	})
	t.Run("using dev_override: tolerates lack of version data when provider not managed by Terraform", func(t *testing.T) {
		// Create a temporary working directory with state store configuration
		// that matches the backend state file
		td := t.TempDir()
		testCopyDir(t, testFixturePath("state-store-unchanged/dev-override-provider/"), td)
		t.Chdir(td)

		mockProvider := mockPluggableStateStorageProvider(mockSingleStateStoreSchema("test_store"))
		// If the working directory was previously initialized successfully then at least
		// one workspace is guaranteed to exist when a user is re-running init with no config
		// changes since last init. So this test says `default` exists.
		mockProvider.GetStatesResponse = &providers.GetStatesResponse{
			States: []string{"default"},
		}
		mockProviderAddress := addrs.NewDefaultProvider("test")
		providerSource := newMockProviderSource(t, map[string][]string{
			"hashicorp/test": {"1.2.3"}, // Matches provider version in backend state file fixture
		})

		ui := testUiWrapped(t)
		view, done := testView(t)
		meta := Meta{
			Ui:                        ui,
			View:                      view,
			AllowExperimentalFeatures: true,
			testingOverrides: &testingOverrides{
				Providers: map[addrs.Provider]providers.Factory{
					mockProviderAddress: providers.FactoryFixed(mockProvider),
				},
			},
			ProviderSource: providerSource,

			// THIS ALLOWS THE TEST TO MIMIC A SCENARIO WHERE THE PROVIDER IS A DEV OVERRIDE.
			// The mock is still accessed via testingOverrides, but Terraform believes that it's a dev override due to
			// its presence in the ProviderDevOverrides map.
			ProviderDevOverrides: map[addrs.Provider]getproviders.PackageLocalDir{
				mockProviderAddress: ".",
			},
		}
		c := &InitCommand{
			Meta: meta,
		}

		// Before running init, confirm the contents of the backend state file before
		statePath := filepath.Join(meta.DataDir(), DefaultStateFilename)
		sMgr := &clistate.LocalState{Path: statePath}
		if err := sMgr.RefreshState(); err != nil {
			t.Fatal("Failed to load state:", err)
		}
		s := sMgr.State()
		if s == nil {
			t.Fatal("expected backend state file to be present, but there isn't one")
		}
		expectedState := &workdir.StateStoreConfigState{
			Type: "test_store",
			Provider: &workdir.ProviderConfigState{
				Version: nil,
				Source: &tfaddr.Provider{
					Hostname:  tfaddr.DefaultProviderRegistryHost,
					Namespace: "hashicorp",
					Type:      "test",
				},
				ConfigRaw: []byte("{\n                \"region\": null\n            }"),
			},
			ConfigRaw:          []byte("{\n            \"value\": \"foobar\"\n        }"),
			Hash:               uint64(3434781367),
			ProviderSupplyMode: getproviders.DevOverride,
		}
		if diff := cmp.Diff(s.StateStore, expectedState); diff != "" {
			t.Fatalf("unexpected diff in backend state file's description of state store:\n%s", diff)
		}

		// Run init command
		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 0 {
			t.Fatalf("expected code 0 exit code, got %d, output: \n%s", code, testOutput.All())
		}

		// Check output
		output := testOutput.All()
		expectedOutputs := []string{
			`Initializing the state store "test_store"...`,
			"Terraform has been successfully initialized!",
		}
		for _, expected := range expectedOutputs {
			if !strings.Contains(output, expected) {
				t.Fatalf("expected output to include %q, but got':\n %s", expected, output)
			}
		}

		// Confirm init was a no-op and backend state is unchanged afterwards
		if err := sMgr.RefreshState(); err != nil {
			t.Fatal("Failed to load state:", err)
		}
		s = sMgr.State()
		if diff := cmp.Diff(s.StateStore, expectedState); diff != "" {
			t.Fatalf("unexpected diff in backend state file's description of state store:\n%s", diff)
		}
	})
}

// Testing init's behaviors with `state_store` when run in a working directory where the configuration
// doesn't match the backend state file.
func TestInit_stateStore_configChanges(t *testing.T) {
	t.Run("the -reconfigure flag makes Terraform ignore the backend state file during initialization", func(t *testing.T) {
		// Create a temporary working directory with state store configuration
		// that doesn't match the backend state file
		td := t.TempDir()
		testCopyDir(t, testFixturePath("state-store-changed/store-config"), td)
		t.Chdir(td)

		mockProvider := mockPluggableStateStorageProvider(mockSingleStateStoreSchema("test_store"))
		// The previous init implied by this test scenario would have created this.
		mockProvider.MockStates = testing_provider.NewMockStateBytesWithSingleState(
			"test_store",
			"default",
			[]byte(`{"version": 4,"terraform_version":"1.15.0","serial": 1,"lineage": "","outputs": {},"resources": [],"checks":[]}`),
		)

		mockProviderAddress := addrs.NewDefaultProvider("test")
		providerSource := newMockProviderSource(t, map[string][]string{
			"hashicorp/test": {"1.2.3"}, // Matches provider version in backend state file fixture
		})

		ui := testUiWrapped(t)
		view, done := testView(t)
		meta := Meta{
			Ui:                        ui,
			View:                      view,
			AllowExperimentalFeatures: true,
			testingOverrides: &testingOverrides{
				Providers: map[addrs.Provider]providers.Factory{
					mockProviderAddress: providers.FactoryFixed(mockProvider),
				},
			},
			ProviderSource: providerSource,
		}
		c := &InitCommand{
			Meta: meta,
		}

		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
			"-reconfigure",
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 0 {
			t.Fatalf("expected code 0 exit code, got %d, output: \n%s", code, testOutput.All())
		}

		// Check output
		output := testOutput.All()
		expectedOutputs := []string{
			`Initializing the state store "test_store"...`,
			"Terraform has been successfully initialized!",
		}
		for _, expected := range expectedOutputs {
			if !strings.Contains(output, expected) {
				t.Fatalf("expected output to include %q, but got':\n %s", expected, output)
			}
		}

		// Assert contents of the backend state file
		statePath := filepath.Join(meta.DataDir(), DefaultStateFilename)
		sMgr := &clistate.LocalState{Path: statePath}
		if err := sMgr.RefreshState(); err != nil {
			t.Fatal("Failed to load state:", err)
		}
		s := sMgr.State()
		if s == nil {
			t.Fatal("expected backend state file to be created, but there isn't one")
		}
		v1_2_3, _ := version.NewVersion("1.2.3")
		expectedState := &workdir.StateStoreConfigState{
			Type:               "test_store",
			ConfigRaw:          []byte("{\n      \"value\": \"changed-value\"\n    }"),
			Hash:               uint64(1157855489), // The new hash after reconfiguring; this doesn't match the backend state test fixture
			ProviderSupplyMode: getproviders.ManagedByTerraform,
			Provider: &workdir.ProviderConfigState{
				Version: v1_2_3,
				Source: &tfaddr.Provider{
					Hostname:  tfaddr.DefaultProviderRegistryHost,
					Namespace: "hashicorp",
					Type:      "test",
				},
				ConfigRaw: []byte("{\n        \"region\": null\n      }"),
			},
		}
		if diff := cmp.Diff(s.StateStore, expectedState); diff != "" {
			t.Fatalf("unexpected diff in backend state file's description of state store:\n%s", diff)
		}
	})

	t.Run("the -backend=false flag makes Terraform ignore config and use only the the backend state file during initialization", func(t *testing.T) {
		// Create a temporary working directory with state store configuration
		// that doesn't match the backend state file
		td := t.TempDir()
		testCopyDir(t, testFixturePath("state-store-changed/store-config"), td)
		t.Chdir(td)

		mockProvider := mockPluggableStateStorageProvider(mockSingleStateStoreSchema("test_store"))
		mockProvider.MockStates = testing_provider.NewMockStateBytesWithStateIds("test_store", []string{"default"})

		// The previous init implied by this test scenario would have created this.
		mockProvider.GetStatesResponse = &providers.GetStatesResponse{States: []string{"default"}}
		mockProvider.MockStates = testing_provider.NewMockStateBytesWithSingleState(
			"test_store",
			"default",
			[]byte(`{"version": 4,"terraform_version":"1.15.0","serial": 1,"lineage": "","outputs": {},"resources": [],"checks":[]}`),
		)

		mockProviderAddress := addrs.NewDefaultProvider("test")
		providerSource := newMockProviderSource(t, map[string][]string{
			"hashicorp/test": {"1.2.3"}, // Matches provider version in backend state file fixture
		})

		ui := testUiWrapped(t)
		view, done := testView(t)
		meta := Meta{
			Ui:                        ui,
			View:                      view,
			AllowExperimentalFeatures: true,
			testingOverrides: &testingOverrides{
				Providers: map[addrs.Provider]providers.Factory{
					mockProviderAddress: providers.FactoryFixed(mockProvider),
				},
			},
			ProviderSource: providerSource,
		}
		c := &InitCommand{
			Meta: meta,
		}

		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
			"-backend=false",
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 0 {
			t.Fatalf("expected code 0 exit code, got %d, output: \n%s", code, testOutput.All())
		}

		// Check output
		output := testOutput.All()
		expectedOutput := "Terraform has been successfully initialized!"
		if !strings.Contains(output, expectedOutput) {
			t.Fatalf("expected output to include %q, but got':\n %s", expectedOutput, output)
		}

		// When -backend=false the backend/state store isn't initialized, so we don't expect this
		// output if the flag has the expected effect on Terraform.
		unexpectedOutput := `Initializing the state store "test_store"...`
		if strings.Contains(output, unexpectedOutput) {
			t.Fatalf("output included %q, which is unexpected if -backend=false is behaving correctly':\n %s", unexpectedOutput, output)
		}
	})

	t.Run("an error is returned from init regardless of -migrate-state flag missing or present", func(t *testing.T) {
		// Create a temporary working directory with state store configuration
		// that doesn't match the backend state file
		td := t.TempDir()
		testCopyDir(t, testFixturePath("state-store-changed/store-config"), td)
		t.Chdir(td)

		mockProvider := mockPluggableStateStorageProvider(mockSingleStateStoreSchema("test_store"))
		mockProvider.MockStates = testing_provider.NewMockStateBytesWithStateIds("test_store", []string{"default"}) // The previous init implied by this test scenario would have created the default workspace.
		mockProviderAddress := addrs.NewDefaultProvider("test")
		providerSource := newMockProviderSource(t, map[string][]string{
			"hashicorp/test": {"1.2.3"}, // Matches provider version in backend state file fixture
		})

		ui := testUiWrapped(t)
		view, done := testView(t)
		meta := Meta{
			Ui:                        ui,
			View:                      view,
			AllowExperimentalFeatures: true,
			testingOverrides: &testingOverrides{
				Providers: map[addrs.Provider]providers.Factory{
					mockProviderAddress: providers.FactoryFixed(mockProvider),
				},
			},
			ProviderSource: providerSource,
		}
		c := &InitCommand{
			Meta: meta,
		}

		// 1) When flag is missing
		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
			// missing -migrate-state flag
		}
		code := c.Run(args)
		testOutput := done(t)
		if code == 0 {
			t.Fatalf("expected non-zero exit code, got %d, output: \n%s", code, testOutput.All())
		}

		// Check output
		expectedErrMsgs := []string{
			"Error: State store initialization required, please run \"terraform state migrate\" or \"terraform init -reconfigure\"",
			`Reason: State store "test_store" (hashicorp/test) configuration changed`,
		}
		output := cleanString(testOutput.Stderr())
		for _, expectedErr := range expectedErrMsgs {
			if !strings.Contains(output, expectedErr) {
				t.Fatalf("unexpected error, expected %q, given: %q", expectedErr, output)
			}
		}

		// 2) When flag is present
		ui = testUiWrapped(t)
		view, done = testView(t)
		meta.Ui = ui
		meta.View = view
		c.Meta = meta

		args = []string{
			"-enable-pluggable-state-storage-experiment=true",
			"-migrate-state",
		}
		code = c.Run(args)
		testOutput = done(t)
		if code == 0 {
			t.Fatalf("expected non-zero exit code, got %d, output: \n%s", code, testOutput.All())
		}

		// Check output - should be same as before
		output = cleanString(testOutput.Stderr())
		for _, expectedErr := range expectedErrMsgs {
			if !strings.Contains(output, expectedErr) {
				t.Fatalf("unexpected error, expected %q, given: %q", expectedErr, output)
			}
		}
	})

	t.Run("dev_override: changes in state store config are reported ok despite lack of version data", func(t *testing.T) {
		// Create a temporary working directory with state store configuration
		// that doesn't match the backend state file
		td := t.TempDir()
		testCopyDir(t, testFixturePath("state-store-changed/store-config-with-dev-override"), td)
		t.Chdir(td)

		mockProvider := mockPluggableStateStorageProvider(mockSingleStateStoreSchema("test_store"))
		mockProviderAddress := addrs.NewDefaultProvider("test")
		providerSource := newMockProviderSource(t, map[string][]string{
			"hashicorp/test": {"1.2.3"}, // Matches provider version in backend state file fixture
		})

		ui := testUiWrapped(t)
		view, done := testView(t)
		meta := Meta{
			Ui:                        ui,
			View:                      view,
			AllowExperimentalFeatures: true,
			testingOverrides: &testingOverrides{
				Providers: map[addrs.Provider]providers.Factory{
					mockProviderAddress: providers.FactoryFixed(mockProvider),
				},
			},
			ProviderSource: providerSource,
			// THIS ALLOWS THE TEST TO MIMIC A SCENARIO WHERE THE PROVIDER IS A DEV OVERRIDE.
			// The mock is still accessed via testingOverrides, but Terraform believes that it's a dev override due to
			// its presence in the ProviderDevOverrides map.
			ProviderDevOverrides: map[addrs.Provider]getproviders.PackageLocalDir{
				mockProviderAddress: ".",
			},
		}
		c := &InitCommand{
			Meta: meta,
		}

		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
			// Just a plan init command, which should return early due to a config change being detected
		}
		code := c.Run(args)
		testOutput := done(t)
		if code == 0 {
			t.Fatalf("expected non-zero exit code, got %d, output: \n%s", code, testOutput.All())
		}

		// Check output
		expectedErrMsgs := []string{
			"Error: State store initialization required, please run \"terraform state migrate\" or \"terraform init -reconfigure\"",
			`Reason: State store "test_store" (hashicorp/test) configuration changed`,
		}
		output := cleanString(testOutput.Stderr())
		for _, expectedErr := range expectedErrMsgs {
			if !strings.Contains(output, expectedErr) {
				t.Fatalf("unexpected error, expected %q, given: %q", expectedErr, output)
			}
		}
	})

	// TODO: once state migrate command is implemented, add test cases in state_migrate_test.go matching the tests
	// removed from this file in the same commit as this TODO.
}

// Test all scenarios where the init command should detect a change in how a state store is configured or used,
// and ensure that Terraform returns an error and doesn't proceed with initialization.
//
// Each test will also assert that the reason for the error matches expectations.
func TestInit_stateStore_changesDetected(t *testing.T) {
	expectedError := `Error: State store initialization required, please run "terraform state migrate" or "terraform init -reconfigure"`
	t.Run("store config changed", func(t *testing.T) {
		td := t.TempDir()
		testCopyDir(t, testFixturePath("state-store-changed/store-config"), td)
		t.Chdir(td)

		mockProvider := mockPluggableStateStorageProvider(mockSingleStateStoreSchema("test_store"))
		mockProviderAddress := addrs.NewDefaultProvider("test")
		providerSource := newMockProviderSource(t, map[string][]string{
			"hashicorp/test": {"1.2.3"}, // Matches provider version in backend state file fixture
		})

		ui := testUiWrapped(t)
		view, done := testView(t)
		c := &InitCommand{
			Meta: Meta{
				testingOverrides: &testingOverrides{
					Providers: map[addrs.Provider]providers.Factory{
						mockProviderAddress: providers.FactoryFixed(mockProvider),
					},
				},
				ProviderSource:            providerSource,
				Ui:                        ui,
				View:                      view,
				AllowExperimentalFeatures: true,
			},
		}

		args := []string{
			"-enable-pluggable-state-storage-experiment",
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 1 {
			t.Fatalf("Expected Terraform to return an error after detecting need for a state migration, but it was missing: \n%s", testOutput.All())
		}

		if !strings.Contains(testOutput.All(), expectedError) {
			t.Fatalf("expected an error due to Terraform detecting 'State store initialization required', got: %s", testOutput.All())
		}
		expectedReason := `Reason: State store "test_store" (hashicorp/test) configuration changed`
		if !strings.Contains(testOutput.All(), expectedReason) {
			t.Fatalf("expected reason to be: %s\ngot: %s", expectedReason, testOutput.All())
		}
	})
	t.Run("state store provider config changed", func(t *testing.T) {
		td := t.TempDir()
		testCopyDir(t, testFixturePath("state-store-changed/provider-config"), td)
		t.Chdir(td)

		mockProvider := mockPluggableStateStorageProvider(mockSingleStateStoreSchema("test_store"))
		mockProviderAddress := addrs.NewDefaultProvider("test")
		providerSource := newMockProviderSource(t, map[string][]string{
			"hashicorp/test": {"1.2.3"}, // Matches provider version in backend state file fixture
		})

		ui := testUiWrapped(t)
		view, done := testView(t)
		c := &InitCommand{
			Meta: Meta{
				testingOverrides: &testingOverrides{
					Providers: map[addrs.Provider]providers.Factory{
						mockProviderAddress: providers.FactoryFixed(mockProvider),
					},
				},
				ProviderSource:            providerSource,
				Ui:                        ui,
				View:                      view,
				AllowExperimentalFeatures: true,
			},
		}

		args := []string{
			"-enable-pluggable-state-storage-experiment",
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 1 {
			t.Fatalf("Expected Terraform to return an error after detecting need for a state migration, but it was missing: \n%s", testOutput.All())
		}

		if !strings.Contains(testOutput.All(), expectedError) {
			t.Fatalf("expected an error due to Terraform detecting 'State store initialization required', got: %s", testOutput.All())
		}
		expectedReason := `Reason: State store provider "test" (hashicorp/test) configuration changed`
		if !strings.Contains(testOutput.All(), expectedReason) {
			t.Fatalf("expected reason to be: %s\ngot: %s", expectedReason, testOutput.All())
		}
	})
	t.Run("state store type changed, within same provider", func(t *testing.T) {
		td := t.TempDir()
		testCopyDir(t, testFixturePath("state-store-changed/state-store-type"), td)
		t.Chdir(td)

		mockProvider := mockPluggableStateStorageProvider(map[string]providers.Schema{
			"test_store": {
				Body: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"value": {
							Type:     cty.String,
							Required: true,
						},
					},
				},
			},
			"test_otherstore": {
				Body: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"value": {
							Type:     cty.String,
							Required: true,
						},
					},
				},
			},
		})
		mockProviderAddress := addrs.NewDefaultProvider("test")
		providerSource := newMockProviderSource(t, map[string][]string{
			"hashicorp/test": {"1.2.3"}, // Matches provider version in backend state file fixture
		})

		ui := testUiWrapped(t)
		view, done := testView(t)
		c := &InitCommand{
			Meta: Meta{
				testingOverrides: &testingOverrides{
					Providers: map[addrs.Provider]providers.Factory{
						mockProviderAddress: providers.FactoryFixed(mockProvider),
					},
				},
				ProviderSource:            providerSource,
				Ui:                        ui,
				View:                      view,
				AllowExperimentalFeatures: true,
			},
		}

		args := []string{
			"-enable-pluggable-state-storage-experiment",
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 1 {
			t.Fatalf("Expected Terraform to return an error after detecting need for a state migration, but it was missing: \n%s", testOutput.All())
		}

		if !strings.Contains(testOutput.All(), expectedError) {
			t.Fatalf("expected an error due to Terraform detecting 'State store initialization required', got: %s", testOutput.All())
		}
		expectedReason := `Reason: State store type changed from "test_store" to "test_otherstore"`
		if !strings.Contains(testOutput.All(), expectedReason) {
			t.Fatalf("expected reason to be: %s\ngot: %s", expectedReason, testOutput.All())
		}
	})
	t.Run("state store provider changed", func(t *testing.T) {
		td := t.TempDir()
		testCopyDir(t, testFixturePath("state-store-changed/provider-used"), td)
		t.Chdir(td)

		providerSource := newMockProviderSource(t, map[string][]string{
			"hashicorp/test":  {"1.2.3"}, // Matches provider version in backend state file fixture
			"hashicorp/test2": {"1.2.3"}, // Matches provider version in backend state file fixture
		})

		ui := testUiWrapped(t)
		view, done := testView(t)
		c := &InitCommand{
			Meta: Meta{
				testingOverrides: &testingOverrides{
					Providers: map[addrs.Provider]providers.Factory{
						addrs.NewDefaultProvider("test"):  providers.FactoryFixed(mockPluggableStateStorageProvider(mockSingleStateStoreSchema("test_src"))),
						addrs.NewDefaultProvider("test2"): providers.FactoryFixed(mockPluggableStateStorageProvider(mockSingleStateStoreSchema("test2_dst"))),
					},
				},
				ProviderSource:            providerSource,
				Ui:                        ui,
				View:                      view,
				AllowExperimentalFeatures: true,
			},
		}

		args := []string{
			"-enable-pluggable-state-storage-experiment",
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 1 {
			t.Fatalf("Expected Terraform to return an error after detecting need for a state migration, but it was missing: \n%s", testOutput.All())
		}

		if !strings.Contains(testOutput.All(), expectedError) {
			t.Fatalf("expected an error due to Terraform detecting 'State store initialization required', got: %s", testOutput.All())
		}
		expectedReason := `Reason: State store provider changed from hashicorp/test to hashicorp/test2`
		if !strings.Contains(testOutput.All(), expectedReason) {
			t.Fatalf("expected reason to be: %s\ngot: %s", expectedReason, testOutput.All())
		}
	})

	// This test handles an edge case where the dependency lock file has been changed out of band.
	// Currently it is impossible to change the state store provider version via init and it must be done
	// as part of a state migration via the state migrate command.
	t.Run("state store provider version changed (out of band)", func(t *testing.T) {
		td := t.TempDir()
		testCopyDir(t, testFixturePath("state-store-changed/provider-version-changed"), td)
		t.Chdir(td)

		providerSource := newMockProviderSource(t, map[string][]string{
			"hashicorp/test": {
				"1.2.3", // Matches provider version in backend state file fixture
				"1.5.0", // Matches provider version in lock file fixture
			},
		})

		ui := testUiWrapped(t)
		view, done := testView(t)
		c := &InitCommand{
			Meta: Meta{
				testingOverrides: &testingOverrides{
					Providers: map[addrs.Provider]providers.Factory{
						addrs.NewDefaultProvider("test"): providers.FactoryFixed(mockPluggableStateStorageProvider(mockSingleStateStoreSchema("test_store"))),
					},
				},
				ProviderSource:            providerSource,
				Ui:                        ui,
				View:                      view,
				AllowExperimentalFeatures: true,
			},
		}

		args := []string{
			"-enable-pluggable-state-storage-experiment",
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 1 {
			t.Fatalf("Expected Terraform to return an error after detecting need for a state migration, but it was missing: \n%s", testOutput.All())
		}

		if !strings.Contains(testOutput.All(), expectedError) {
			t.Fatalf("expected an error due to Terraform detecting 'State store initialization required', got: %s", testOutput.All())
		}
		expectedReason := "Reason: State store provider \"test\" (hashicorp/test) version changed from\n1.2.3 to 1.5.0"
		if !strings.Contains(testOutput.All(), expectedReason) {
			t.Fatalf("expected reason to be: %s\ngot: %s", expectedReason, testOutput.All())
		}
	})
}

// Test a scenario where the configuration changes but the -backend-config CLI flags compensate for those changes
// to result in the state store being configured in the same way. In this scenario the user isn't prompted to migrate
// state but the backend state file is updated with a new hash to reflect the new configuration's values.
func TestInit_stateStore_backendConfigFlagNoMigrate(t *testing.T) {
	// Create a temporary working directory and copy in test fixtures
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-state-store"), td)
	t.Chdir(td)

	mockProvider := mockPluggableStateStorageProvider(mockSingleStateStoreSchema("test_store"))
	mockProvider.MockStates = testing_provider.NewMockStateBytesWithStateIds("test_store", []string{"default"})
	// Make the state store's value attribute optional in this test.
	mockProvider.GetProviderSchemaResponse.StateStores["test_store"].Body.Attributes["value"].Required = false

	mockProviderAddress := addrs.NewDefaultProvider("test")
	providerSource := newMockProviderSource(t, map[string][]string{
		"hashicorp/test": {"1.2.3"}, // Matches provider version in backend state file fixture
	})

	var originalStateStoreConfigHash uint64

	{
		log.Printf("[TRACE] TestInit_stateStore_backendConfigFlagNoMigrate: beginning first init")

		ui := testUiWrapped(t)
		view, done := testView(t)
		c := &InitCommand{
			Meta: Meta{
				testingOverrides: &testingOverrides{
					Providers: map[addrs.Provider]providers.Factory{
						mockProviderAddress: providers.FactoryFixed(mockProvider),
					},
				},
				ProviderSource:            providerSource,
				Ui:                        ui,
				View:                      view,
				AllowExperimentalFeatures: true,
			},
		}

		// Init
		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 0 {
			t.Fatalf("bad: \n%s", testOutput.All())
		}
		log.Printf("[TRACE] TestInit_stateStore_backendConfigFlagNoMigrate: first init complete")
		t.Logf("First run output:\n%s", testOutput.Stdout())
		t.Logf("First run errors:\n%s", testOutput.Stderr())

		s := testDataStateRead(t, filepath.Join(DefaultDataDir, DefaultStateFilename))
		if s.StateStore.Empty() {
			t.Fatal("should have StateStore config")
		}
		// Store this for comparison after the 2nd init
		originalStateStoreConfigHash = s.StateStore.Hash
	}

	{
		log.Printf("[TRACE] TestInit_stateStore_backendConfigFlagNoMigrate: beginning second init with changed config but compensating CLI flags")

		// Remove `value` attribute from config
		cfg := `terraform {
  state_store "test_store" {
    provider "test" {}
    # value attr removed here
  }
}`
		if err := os.WriteFile("main.tf", []byte(cfg), 0644); err != nil {
			t.Fatalf("err: %s", err)
		}

		ui := testUiWrapped(t)
		view, done := testView(t)
		c := &InitCommand{
			Meta: Meta{
				testingOverrides: &testingOverrides{
					Providers: map[addrs.Provider]providers.Factory{
						mockProviderAddress: providers.FactoryFixed(mockProvider),
					},
				},
				ProviderSource:            providerSource,
				Ui:                        ui,
				View:                      view,
				AllowExperimentalFeatures: true,
			},
		}

		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
			"-backend-config=value=foobar", // value = foobar, matches the line removed from config
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 0 {
			t.Fatalf("Terraform either experienced an unexpected error, or suggested a state migration when this test scenario should not include migrations: \n%s", testOutput.All())
		}
		log.Printf("[TRACE] TestInit_stateStore_backendConfigFlagNoMigrate: second init complete")
		t.Logf("Second run output:\n%s", testOutput.Stdout())
		t.Logf("Second run errors:\n%s", testOutput.Stderr())

		s := testDataStateRead(t, filepath.Join(DefaultDataDir, DefaultStateFilename))
		if s.StateStore.Empty() {
			t.Fatal("should have StateStore config")
		}
		if s.StateStore.Hash == originalStateStoreConfigHash {
			t.Fatal("expected second init to update the state_store config hash in the backend state file, but it did not")
		}
	}
}

func TestInit_stateStore_unset(t *testing.T) {
	// Create a temporary working directory and copy in test fixtures
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-state-store"), td)
	t.Chdir(td)

	mockProvider := mockPluggableStateStorageProvider(mockSingleStateStoreSchema("test_store"))
	mockProvider.MockStates = testing_provider.NewMockStateBytesWithStateIds("test_store", []string{"default"})
	storeName := "test_store"
	otherStoreName := "test_otherstore"
	// Make the provider report that it contains a 2nd storage implementation with the above name
	mockProvider.GetProviderSchemaResponse.StateStores[otherStoreName] = mockProvider.GetProviderSchemaResponse.StateStores[storeName]
	mockProviderAddress := addrs.NewDefaultProvider("test")
	providerSource := newMockProviderSource(t, map[string][]string{
		"hashicorp/test": {"1.2.3"}, // Matches provider version in backend state file fixture
	})

	{
		log.Printf("[TRACE] TestInit_stateStore_unset: beginning first init")

		ui := testUiWrapped(t)
		view, done := testView(t)
		c := &InitCommand{
			Meta: Meta{
				testingOverrides: &testingOverrides{
					Providers: map[addrs.Provider]providers.Factory{
						mockProviderAddress: providers.FactoryFixed(mockProvider),
					},
				},
				ProviderSource:            providerSource,
				Ui:                        ui,
				View:                      view,
				AllowExperimentalFeatures: true,
			},
		}

		// Init
		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 0 {
			t.Fatalf("bad: \n%s", testOutput.All())
		}
		log.Printf("[TRACE] TestInit_stateStore_unset: first init complete")
		t.Logf("First run output:\n%s", testOutput.Stdout())
		t.Logf("First run errors:\n%s", testOutput.Stderr())

		if _, err := os.Stat(filepath.Join(DefaultDataDir, DefaultStateFilename)); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		log.Printf("[TRACE] TestInit_stateStore_unset: beginning second init")

		// Unset
		if err := os.WriteFile("main.tf", []byte(""), 0644); err != nil {
			t.Fatalf("err: %s", err)
		}

		ui := testUiWrapped(t)
		view, done := testView(t)
		c := &InitCommand{
			Meta: Meta{
				testingOverrides: &testingOverrides{
					Providers: map[addrs.Provider]providers.Factory{
						mockProviderAddress: providers.FactoryFixed(mockProvider),
					},
				},
				ProviderSource:            providerSource,
				Ui:                        ui,
				View:                      view,
				AllowExperimentalFeatures: true,
			},
		}

		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
			"-force-copy",
		}
		code := c.Run(args)
		testOutput := done(t)
		if code == 0 {
			t.Fatalf("expected non-zero exit code, got 0: \n%s", testOutput.All())
		}
		log.Printf("[TRACE] TestInit_stateStore_unset: second init complete")
		expectedMsgs := []string{
			"Error: State store initialization required, please run \"terraform state migrate\" or \"terraform init -reconfigure\"",
			"Reason: Unsetting the previously set state store \"test_store\"",
		}
		output := cleanString(testOutput.Stderr())
		for _, expectedMsg := range expectedMsgs {
			if !strings.Contains(output, expectedMsg) {
				t.Fatalf("expected error message %q not found: \n%s", expectedMsg, output)
			}
		}
	}

	// TODO: Create a test in state_migrate_test.go where the terraform state migrate command is used for the migration,
	// and assert that after migration the local state contains the expected state.
}

func TestInit_stateStore_unset_withoutProviderRequirements(t *testing.T) {
	// Create a temporary working directory and copy in test fixtures
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-state-store"), td)
	t.Chdir(td)

	mockProvider := mockPluggableStateStorageProvider(mockSingleStateStoreSchema("test_store"))
	mockProvider.MockStates = testing_provider.NewMockStateBytesWithStateIds("test_store", []string{"default"})
	mockProviderAddress := addrs.NewDefaultProvider("test")
	providerSource := newMockProviderSource(t, map[string][]string{
		"hashicorp/test": {"1.2.3"}, // Matches provider version in backend state file fixture
	})

	{
		log.Printf("[TRACE] TestInit_stateStore_unset_withoutProviderRequirements: beginning first init")

		ui := testUiWrapped(t)
		view, done := testView(t)
		c := &InitCommand{
			Meta: Meta{
				testingOverrides: &testingOverrides{
					Providers: map[addrs.Provider]providers.Factory{
						mockProviderAddress: providers.FactoryFixed(mockProvider),
					},
				},
				ProviderSource:            providerSource,
				Ui:                        ui,
				View:                      view,
				AllowExperimentalFeatures: true,
			},
		}

		// Init
		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 0 {
			t.Fatalf("bad: \n%s", testOutput.All())
		}
		log.Printf("[TRACE] TestInit_stateStore_unset_withoutProviderRequirements: first init complete")
		t.Logf("First run output:\n%s", testOutput.Stdout())
		t.Logf("First run errors:\n%s", testOutput.Stderr())

		if _, err := os.Stat(filepath.Join(DefaultDataDir, DefaultStateFilename)); err != nil {
			t.Fatalf("err: %s", err)
		}
	}
	{
		log.Printf("[TRACE] TestInit_stateStore_unset_withoutProviderRequirements: beginning second init")
		// Unset state store and provider requirements
		if err := os.WriteFile("main.tf", []byte(""), 0644); err != nil {
			t.Fatalf("err: %s", err)
		}
		if err := os.WriteFile("providers.tf", []byte(""), 0644); err != nil {
			t.Fatalf("err: %s", err)
		}

		ui := testUiWrapped(t)
		view, done := testView(t)
		c := &InitCommand{
			Meta: Meta{
				testingOverrides: &testingOverrides{
					Providers: map[addrs.Provider]providers.Factory{
						mockProviderAddress: providers.FactoryFixed(mockProvider),
					},
				},
				ProviderSource:            providerSource,
				Ui:                        ui,
				View:                      view,
				AllowExperimentalFeatures: true,
			},
		}

		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
			"-force-copy",
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 1 {
			t.Fatalf("expected code 1 exit code, got %d, output: \n%s", code, testOutput.All())
		}

		expectedErrMsgs := []string{
			"Error: State store initialization required, please run \"terraform state migrate\" or \"terraform init -reconfigure\"",
			"Reason: Unsetting the previously set state store \"test_store\"",
		}
		output := cleanString(testOutput.Stderr())
		for _, expectedErr := range expectedErrMsgs {
			if !strings.Contains(output, expectedErr) {
				t.Fatalf("unexpected error, expected %q, given: %q", expectedErr, output)
			}
		}
	}
}

func TestInit_stateStore_to_backend(t *testing.T) {
	// Create a temporary working directory and copy in test fixtures
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-state-store"), td)
	t.Chdir(td)

	mockProvider := mockPluggableStateStorageProvider(mockSingleStateStoreSchema("test_store"))
	mockProvider.MockStates = testing_provider.NewMockStateBytesWithStateIds("test_store", []string{"default"})
	mockProviderAddress := addrs.NewDefaultProvider("test")
	providerSource := newMockProviderSource(t, map[string][]string{
		"hashicorp/test": {"1.2.3"}, // Matches provider version in backend state file fixture
	})

	tOverrides := &testingOverrides{
		Providers: map[addrs.Provider]providers.Factory{
			mockProviderAddress: providers.FactoryFixed(mockProvider),
		},
	}

	{
		log.Printf("[TRACE] TestInit_stateStore_to_backend: beginning first init")
		// Init
		ui := testUiWrapped(t)
		view, done := testView(t)
		c := &InitCommand{
			Meta: Meta{
				testingOverrides:          tOverrides,
				ProviderSource:            providerSource,
				Ui:                        ui,
				View:                      view,
				AllowExperimentalFeatures: true,
			},
		}
		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 0 {
			t.Fatalf("bad: \n%s", testOutput.All())
		}
		log.Printf("[TRACE] TestInit_stateStore_to_backend: first init complete")
		t.Logf("First run output:\n%s", testOutput.Stdout())
		t.Logf("First run errors:\n%s", testOutput.Stderr())

		if _, err := os.Stat(filepath.Join(DefaultDataDir, DefaultStateFilename)); err != nil {
			t.Fatalf("error when checking the backend state file exists: %s", err)
		}
	}
	{
		// run apply to ensure state isn't empty
		// to bypass edge case handling which causes empty state to stop migration
		log.Printf("[TRACE] TestInit_stateStore_to_backend: beginning apply")
		ui := testUiWrapped(t)
		aView, aDone := testView(t)
		cApply := &ApplyCommand{
			Meta: Meta{
				testingOverrides:          tOverrides,
				ProviderSource:            providerSource,
				Ui:                        ui,
				View:                      aView,
				AllowExperimentalFeatures: true,
			},
		}
		aCode := cApply.Run([]string{"-auto-approve"})
		aTestOutput := aDone(t)
		if aCode != 0 {
			t.Fatalf("bad: \n%s", aTestOutput.All())
		}

		t.Logf("Apply output:\n%s", aTestOutput.Stdout())
		t.Logf("Apply errors:\n%s", aTestOutput.Stderr())
	}
	{
		log.Printf("[TRACE] TestInit_stateStore_to_backend: beginning uninitialised apply")

		backendCfg := []byte(`terraform {
  backend "http" {
    address = "https://example.com"
  }
}
`)
		if err := os.WriteFile("main.tf", backendCfg, 0644); err != nil {
			t.Fatalf("err: %s", err)
		}

		ui := testUiWrapped(t)
		view, done := testView(t)
		cApply := &ApplyCommand{
			Meta: Meta{
				testingOverrides:          tOverrides,
				ProviderSource:            providerSource,
				Ui:                        ui,
				View:                      view,
				AllowExperimentalFeatures: true,
			},
		}
		code := cApply.Run([]string{"-auto-approve"})
		testOutput := done(t)
		if code == 0 {
			t.Fatalf("expected apply to fail: \n%s", testOutput.All())
		}
		log.Printf("[TRACE] TestInit_stateStore_to_backend: apply complete")
		expectedErr := "State store initialization required"
		if !strings.Contains(testOutput.Stderr(), expectedErr) {
			t.Fatalf("unexpected error, expected %q, given: %q", expectedErr, testOutput.Stderr())
		}

		log.Printf("[TRACE] TestInit_stateStore_to_backend: uninitialised apply complete")
		t.Logf("First run output:\n%s", testOutput.Stdout())
		t.Logf("First run errors:\n%s", testOutput.Stderr())
	}
	{
		log.Printf("[TRACE] TestInit_stateStore_to_backend: beginning second init")

		testBackend := new(httpBackend.TestHTTPBackend)
		ts := httptest.NewServer(http.HandlerFunc(testBackend.Handle))
		t.Cleanup(ts.Close)

		// Override state store to backend
		backendCfg := fmt.Sprintf(`terraform {
  backend "http" {
    address = %q
  }
}
`, ts.URL)
		if err := os.WriteFile("main.tf", []byte(backendCfg), 0644); err != nil {
			t.Fatalf("err: %s", err)
		}

		ui := testUiWrapped(t)
		view, done := testView(t)
		c := &InitCommand{
			Meta: Meta{
				testingOverrides: &testingOverrides{
					Providers: map[addrs.Provider]providers.Factory{
						mockProviderAddress: providers.FactoryFixed(mockProvider),
					},
				},
				ProviderSource:            providerSource,
				Ui:                        ui,
				View:                      view,
				AllowExperimentalFeatures: true,
			},
		}

		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
			"-migrate-state",
			"-force-copy",
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 1 {
			t.Fatalf("expected code 1 exit code, got %d, output: \n%s", code, testOutput.All())
		}
		log.Printf("[TRACE] TestInit_stateStore_to_backend: second init complete")

		expectedErrMsgs := []string{
			"Error: State store initialization required, please run \"terraform state migrate\" or \"terraform init -reconfigure\"",
			// Assert both commands are mentioned
			"run \"terraform state migrate\"",
			"run \"terraform init -reconfigure\"",
		}
		output := cleanString(testOutput.Stderr())
		for _, expectedErr := range expectedErrMsgs {
			if !strings.Contains(output, expectedErr) {
				t.Fatalf("unexpected error, expected %q, given: %q", expectedErr, output)
			}
		}
	}

	// TODO: Create a test in state_migrate_test.go where the terraform state migrate command is used for the migration.
}

// Test that users are shown actionable errors if they try to use a state store in a non-init command
// before running an init operation to download the state storage provider and record it in the dependency lock file.
func TestInit_uninitialized_stateStore(t *testing.T) {
	t.Run("error if working directory isn't initialized before apply", func(t *testing.T) {
		// Create a temporary working directory that is empty
		td := t.TempDir()
		cfg := `terraform {
	  required_providers {
	    test = {
	      source = "hashicorp/test"
	    }
	  }
	  state_store "test_store" {
	    provider "test" {}
	    value = "foobar"
	  }
	}
	`
		if err := os.WriteFile(filepath.Join(td, "main.tf"), []byte(cfg), 0644); err != nil {
			t.Fatalf("err: %s", err)
		}
		t.Chdir(td)

		ui := testUiWrapped(t)
		view, done := testView(t)
		cApply := &ApplyCommand{
			Meta: Meta{
				Ui:                        ui,
				View:                      view,
				AllowExperimentalFeatures: true,
			},
		}
		code := cApply.Run([]string{"-no-color"})
		testOutput := done(t)
		if code == 0 {
			t.Fatalf("expected apply to fail: \n%s", testOutput.All())
		}
		log.Printf("[TRACE] TestInit_stateStore_to_backend: uninitialised apply with state store complete")
		expectedErrMsgs := []string{
			"The provider dependency used for state storage is missing from the lock file despite being present in the current configuration",
			`provider registry.terraform.io/hashicorp/test: required by this configuration but no version is selected`,
		}
		for _, expectedErr := range expectedErrMsgs {
			if !strings.Contains(cleanString(testOutput.Stderr()), expectedErr) {
				t.Fatalf("unexpected error, expected %q, given: %s", expectedErr, testOutput.Stderr())
			}
		}
	})

	t.Run("the error isn't shown if the provider is supplied through reattach config", func(t *testing.T) {
		t.Skip("This is implemented as an E2E test: TestPrimary_stateStore_unmanaged_separatePlan")
	})
}

func TestInit_backend_to_stateStore_singleWorkspace(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()

	testBackend := new(httpBackend.TestHTTPBackend)
	ts := httptest.NewServer(http.HandlerFunc(testBackend.Handle))
	t.Cleanup(ts.Close)

	cfg := fmt.Sprintf(`terraform {
  backend "http" {
    address = %q
  }
}
`, ts.URL)
	if err := os.WriteFile(filepath.Join(td, "main.tf"), []byte(cfg), 0644); err != nil {
		t.Fatalf("err: %s", err)
	}
	t.Chdir(td)

	mockProvider := mockPluggableStateStorageProvider(mockSingleStateStoreSchema("test_store"))
	mockProvider.MockStates = testing_provider.NewMockStateBytesWithStateIds("test_store", []string{"default"})
	mockProviderAddress := addrs.NewDefaultProvider("test")
	providerSource := newMockProviderSource(t, map[string][]string{
		"hashicorp/test": {"1.2.3"},
	})

	tOverrides := &testingOverrides{
		Providers: map[addrs.Provider]providers.Factory{
			mockProviderAddress: providers.FactoryFixed(mockProvider),
		},
	}

	{
		log.Printf("[TRACE] %s: beginning first init with backend", t.Name())
		// Init
		ui := testUiWrapped(t)
		view, done := testView(t)
		c := &InitCommand{
			Meta: Meta{
				Ui:                        ui,
				View:                      view,
				AllowExperimentalFeatures: true,
			},
		}
		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 0 {
			t.Fatalf("bad: \n%s", testOutput.All())
		}
		log.Printf("[TRACE] %s: first init complete", t.Name())
		t.Logf("First run output:\n%s", testOutput.Stdout())
		t.Logf("First run errors:\n%s", testOutput.Stderr())

		if testBackend.CallCount("POST") != 0 {
			t.Fatalf("expected 0 POST calls after init, got %d", testBackend.CallCount("POST"))
		}
		if testBackend.CallCount("GET") != 2 {
			t.Fatalf("expected 2 GET calls after init, got %d", testBackend.CallCount("GET"))
		}
	}
	{
		// run apply to ensure state isn't empty
		// to bypass edge case handling which causes empty state to stop migration
		log.Printf("[TRACE] %s: beginning apply with backend", t.Name())

		outputCfg := `output "test" {
  value = "test"
}
`
		if err := os.WriteFile(filepath.Join(td, "output.tf"), []byte(outputCfg), 0644); err != nil {
			t.Fatalf("err: %s", err)
		}

		ui := testUiWrapped(t)
		aView, aDone := testView(t)
		cApply := &ApplyCommand{
			Meta: Meta{
				Ui:                        ui,
				View:                      aView,
				AllowExperimentalFeatures: true,
			},
		}
		aCode := cApply.Run([]string{"-auto-approve"})
		aTestOutput := aDone(t)
		if aCode != 0 {
			t.Fatalf("bad: \n%s", aTestOutput.All())
		}

		t.Logf("Apply output:\n%s", aTestOutput.Stdout())
		t.Logf("Apply errors:\n%s", aTestOutput.Stderr())

		if testBackend.CallCount("POST") != 1 {
			t.Fatalf("expected 1 POST call after apply, got %d", testBackend.CallCount("POST"))
		}
		if testBackend.CallCount("GET") != 5 {
			t.Fatalf("expected 5 GET calls after apply, got %d", testBackend.CallCount("GET"))
		}
		data, err := statefile.Read(bytes.NewBuffer(testBackend.Data))
		if err != nil {
			t.Fatal(err)
		}
		expectedOutputs := map[string]*states.OutputValue{
			"test": {
				Addr: addrs.AbsOutputValue{
					OutputValue: addrs.OutputValue{
						Name: "test",
					},
				},
				Value: cty.StringVal("test"),
			},
		}
		if diff := cmp.Diff(expectedOutputs, data.State.RootOutputValues); diff != "" {
			t.Fatalf("unexpected data after apply: %s", diff)
		}
	}
	{
		log.Printf("[TRACE] %s: beginning second init with state store", t.Name())

		ssCfg := `terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
  state_store "test_store" {
    provider "test" {}
    value = "foobar"
  }
}
`
		if err := os.WriteFile(filepath.Join(td, "main.tf"), []byte(ssCfg), 0644); err != nil {
			t.Fatalf("err: %s", err)
		}

		ui := testUiWrapped(t)
		view, done := testView(t)
		c := &InitCommand{
			Meta: Meta{
				testingOverrides:          tOverrides,
				ProviderSource:            providerSource,
				Ui:                        ui,
				View:                      view,
				AllowExperimentalFeatures: true,
			},
		}

		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
			"-force-copy",
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 1 {
			t.Fatalf("expected code 1 exit code, got %d, output: \n%s", code, testOutput.All())
		}

		expectedErrMsgs := []string{
			"Error: Backend initialization required, please run \"terraform state migrate\" or \"terraform init -reconfigure\"",
			"Reason: Migrating from backend \"http\" to state store \"test_store\" in provider test (\"registry.terraform.io/hashicorp/test\")",
			"run \"terraform state migrate\"",
			"run \"terraform init -reconfigure\"",
		}
		output := cleanString(testOutput.Stderr())
		for _, expectedErr := range expectedErrMsgs {
			if !strings.Contains(output, expectedErr) {
				t.Fatalf("unexpected error, expected %q, given: %q", expectedErr, output)
			}
		}
	}

	// TODO: Create a test in state_migrate_test.go where the terraform state migrate command is used for the migration,
	// and assert that after migration the state store contains the expected state.
}

// TestInit_backend_to_stateStore_noState tests that given no state
// in the source backend, no state is created in the destination state store
// as a result of the migration.
func TestInit_backend_to_stateStore_noState(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()

	testBackend := new(httpBackend.TestHTTPBackend)
	ts := httptest.NewServer(http.HandlerFunc(testBackend.Handle))
	t.Cleanup(ts.Close)

	cfg := fmt.Sprintf(`terraform {
  backend "http" {
    address = %q
  }
}
`, ts.URL)
	if err := os.WriteFile(filepath.Join(td, "main.tf"), []byte(cfg), 0644); err != nil {
		t.Fatalf("err: %s", err)
	}
	t.Chdir(td)

	mockProvider := mockPluggableStateStorageProvider(mockSingleStateStoreSchema("test_store"))
	mockProvider.MockStates = testing_provider.NewMockStateBytesWithStateIds("test_store", []string{"default"})
	mockProviderAddress := addrs.NewDefaultProvider("test")
	providerSource := newMockProviderSource(t, map[string][]string{
		"hashicorp/test": {"1.2.3"},
	})

	tOverrides := &testingOverrides{
		Providers: map[addrs.Provider]providers.Factory{
			mockProviderAddress: providers.FactoryFixed(mockProvider),
		},
	}
	{
		log.Printf("[TRACE] %s: beginning first init with backend", t.Name())
		// Init
		ui := testUiWrapped(t)
		view, done := testView(t)
		c := &InitCommand{
			Meta: Meta{
				Ui:                        ui,
				View:                      view,
				AllowExperimentalFeatures: true,
			},
		}
		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 0 {
			t.Fatalf("first init exited with non-zero code %d:\n%s", code, testOutput.Stderr())
		}
		log.Printf("[TRACE] %s: first init complete", t.Name())
		t.Logf("First run output:\n%s", testOutput.Stdout())
		t.Logf("First run errors:\n%s", testOutput.Stderr())

		if testBackend.CallCount("POST") != 0 {
			t.Fatalf("expected 0 POST calls after init, got %d", testBackend.CallCount("POST"))
		}
		if testBackend.CallCount("GET") != 2 {
			t.Fatalf("expected 2 GET calls after init, got %d", testBackend.CallCount("GET"))
		}
	}
	{
		log.Printf("[TRACE] %s: beginning second init with state store", t.Name())

		ssCfg := `terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
  state_store "test_store" {
    provider "test" {}
    value = "foobar"
  }
}
`
		if err := os.WriteFile(filepath.Join(td, "main.tf"), []byte(ssCfg), 0644); err != nil {
			t.Fatalf("err: %s", err)
		}

		ui := testUiWrapped(t)
		view, done := testView(t)
		c := &InitCommand{
			Meta: Meta{
				testingOverrides:          tOverrides,
				ProviderSource:            providerSource,
				Ui:                        ui,
				View:                      view,
				AllowExperimentalFeatures: true,
			},
		}

		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
			"-force-copy",
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 1 {
			t.Fatalf("expected second init to exit with code 1, got %d:\n%s", code, testOutput.Stderr())
		}
		expectedErrMsgs := []string{
			"Error: Backend initialization required, please run \"terraform state migrate\" or \"terraform init -reconfigure\"",
			"Reason: Migrating from backend \"http\" to state store \"test_store\" in provider test (\"registry.terraform.io/hashicorp/test\")",
			"run \"terraform state migrate\"",
			"run \"terraform init -reconfigure\"",
		}
		output := cleanString(testOutput.Stderr())
		for _, expectedErr := range expectedErrMsgs {
			if !strings.Contains(output, expectedErr) {
				t.Fatalf("unexpected error, expected %q, given: %q", expectedErr, output)
			}
		}
	}

	// TODO: Create a test in state_migrate_test.go where the terraform state migrate command is used for the migration,
	// and assert that after migration the state store is still empty.
}

func TestInit_localBackend_to_stateStore(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()

	cfg := `terraform {
  backend "local" {}
}
`
	if err := os.WriteFile(filepath.Join(td, "main.tf"), []byte(cfg), 0644); err != nil {
		t.Fatalf("err: %s", err)
	}
	t.Chdir(td)

	mockProvider := mockPluggableStateStorageProvider(mockSingleStateStoreSchema("test_store"))
	mockProvider.MockStates = testing_provider.NewMockStateBytesWithStateIds("test_store", []string{"default"})
	mockProviderAddress := addrs.NewDefaultProvider("test")
	providerSource := newMockProviderSource(t, map[string][]string{
		"hashicorp/test": {"1.2.3"},
	})

	tOverrides := &testingOverrides{
		Providers: map[addrs.Provider]providers.Factory{
			mockProviderAddress: providers.FactoryFixed(mockProvider),
		},
	}
	{
		log.Printf("[TRACE] %s: beginning first init with local backend", t.Name())
		// Init
		ui := testUiWrapped(t)
		view, done := testView(t)
		c := &InitCommand{
			Meta: Meta{
				Ui:                        ui,
				View:                      view,
				AllowExperimentalFeatures: true,
			},
		}
		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 0 {
			t.Fatalf("first init exited with non-zero code %d:\n%s", code, testOutput.Stderr())
		}
		log.Printf("[TRACE] %s: first init complete", t.Name())
		t.Logf("First run output:\n%s", testOutput.Stdout())
		t.Logf("First run errors:\n%s", testOutput.Stderr())
	}
	{
		// run apply to ensure state isn't empty
		// to bypass edge case handling which causes empty state to stop migration
		log.Printf("[TRACE] %s: beginning apply with backend", t.Name())

		outputCfg := `output "test" {
  value = "test"
}
`
		if err := os.WriteFile(filepath.Join(td, "output.tf"), []byte(outputCfg), 0644); err != nil {
			t.Fatalf("err: %s", err)
		}

		ui := testUiWrapped(t)
		aView, aDone := testView(t)
		cApply := &ApplyCommand{
			Meta: Meta{
				Ui:                        ui,
				View:                      aView,
				AllowExperimentalFeatures: true,
			},
		}
		aCode := cApply.Run([]string{"-auto-approve"})
		aTestOutput := aDone(t)
		if aCode != 0 {
			t.Fatalf("bad: \n%s", aTestOutput.All())
		}

		t.Logf("Apply output:\n%s", aTestOutput.Stdout())
		t.Logf("Apply errors:\n%s", aTestOutput.Stderr())

		b, err := os.ReadFile(DefaultStateFilename)
		if err != nil {
			t.Fatalf("unable to read state file: %s", err)
		}

		data, err := statefile.Read(bytes.NewBuffer(b))
		if err != nil {
			t.Fatal(err)
		}
		expectedOutputs := map[string]*states.OutputValue{
			"test": {
				Addr: addrs.AbsOutputValue{
					OutputValue: addrs.OutputValue{
						Name: "test",
					},
				},
				Value: cty.StringVal("test"),
			},
		}
		if diff := cmp.Diff(expectedOutputs, data.State.RootOutputValues); diff != "" {
			t.Fatalf("unexpected data after apply: %s", diff)
		}
	}
	{
		log.Printf("[TRACE] %s: beginning second init with state store", t.Name())

		ssCfg := `terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
  state_store "test_store" {
    provider "test" {}
    value = "foobar"
  }
}
`
		if err := os.WriteFile(filepath.Join(td, "main.tf"), []byte(ssCfg), 0644); err != nil {
			t.Fatalf("err: %s", err)
		}

		ui := testUiWrapped(t)
		view, done := testView(t)
		c := &InitCommand{
			Meta: Meta{
				testingOverrides:          tOverrides,
				ProviderSource:            providerSource,
				Ui:                        ui,
				View:                      view,
				AllowExperimentalFeatures: true,
			},
		}

		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
			"-force-copy",
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 1 {
			t.Fatalf("expected second init to exit with code 1, got %d:\n%s", code, testOutput.Stderr())
		}
		expectedErrMsgs := []string{
			"Error: Backend initialization required, please run \"terraform state migrate\" or \"terraform init -reconfigure\"",
			"Reason: Migrating from backend \"local\" to state store \"test_store\" in provider test (\"registry.terraform.io/hashicorp/test\")",
			"run \"terraform state migrate\"",
			"run \"terraform init -reconfigure\"",
		}
		output := cleanString(testOutput.Stderr())
		for _, expectedErr := range expectedErrMsgs {
			if !strings.Contains(output, expectedErr) {
				t.Fatalf("unexpected error, expected %q, given: %q", expectedErr, output)
			}
		}
	}

	// TODO: Create a test in state_migrate_test.go where the terraform state migrate command is used for the migration,
	// and assert that after migration the state store contains the expected state and the local copies are removed.
}

func TestInit_backend_to_stateStore_multipleWorkspaces(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()

	cfg := `terraform {
  backend "inmem" {}
}
`
	if err := os.WriteFile(filepath.Join(td, "main.tf"), []byte(cfg), 0644); err != nil {
		t.Fatalf("err: %s", err)
	}
	t.Chdir(td)

	mockProvider := mockPluggableStateStorageProvider(mockSingleStateStoreSchema("test_store"))
	mockProvider.MockStates = testing_provider.NewMockStateBytesWithStateIds("test_store", []string{"default"})
	mockProviderAddress := addrs.NewDefaultProvider("test")
	providerSource := newMockProviderSource(t, map[string][]string{
		"hashicorp/test": {"1.2.3"},
	})

	tOverrides := &testingOverrides{
		Providers: map[addrs.Provider]providers.Factory{
			mockProviderAddress: providers.FactoryFixed(mockProvider),
		},
	}

	{
		log.Printf("[TRACE] %s: beginning first init with backend", t.Name())
		// Init
		ui := testUiWrapped(t)
		view, done := testView(t)
		c := &InitCommand{
			Meta: Meta{
				Ui:                        ui,
				View:                      view,
				AllowExperimentalFeatures: true,
			},
		}
		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 0 {
			t.Fatalf("bad: \n%s", testOutput.All())
		}
		log.Printf("[TRACE] %s: first init complete", t.Name())
		t.Logf("First run output:\n%s", testOutput.Stdout())
		t.Logf("First run errors:\n%s", testOutput.Stderr())
	}
	{
		// run apply to ensure state isn't empty
		// to bypass edge case handling which causes empty state to stop migration
		log.Printf("[TRACE] %s: beginning first apply to default workspace with backend", t.Name())

		outputCfg := `output "test" {
  value = "test"
}
`
		if err := os.WriteFile(filepath.Join(td, "output.tf"), []byte(outputCfg), 0644); err != nil {
			t.Fatalf("err: %s", err)
		}

		ui := testUiWrapped(t)
		aView, aDone := testView(t)
		cApply := &ApplyCommand{
			Meta: Meta{
				Ui:                        ui,
				View:                      aView,
				AllowExperimentalFeatures: true,
			},
		}
		aCode := cApply.Run([]string{"-auto-approve"})
		aTestOutput := aDone(t)
		if aCode != 0 {
			t.Fatalf("bad: \n%s", aTestOutput.All())
		}

		t.Logf("Apply output:\n%s", aTestOutput.Stdout())
		t.Logf("Apply errors:\n%s", aTestOutput.Stderr())

		data := inmem.ReadState(t, backend.DefaultStateName)
		expectedOutputs := map[string]*states.OutputValue{
			"test": {
				Addr: addrs.AbsOutputValue{
					OutputValue: addrs.OutputValue{
						Name: "test",
					},
				},
				Value: cty.StringVal("test"),
			},
		}
		if diff := cmp.Diff(expectedOutputs, data.RootOutputValues); diff != "" {
			t.Fatalf("unexpected data after apply: %s", diff)
		}
	}
	{
		ui := testUiWrapped(t)
		aView, aDone := testView(t)
		cSelect := &WorkspaceSelectCommand{
			Meta: Meta{
				Ui:                        ui,
				View:                      aView,
				AllowExperimentalFeatures: true,
			},
		}
		sCode := cSelect.Run([]string{"-or-create", "second"})
		aTestOutput := aDone(t)
		if sCode != 0 {
			t.Fatalf("unable to select workspace: \n%s", aTestOutput.All())
		}
		t.Logf("Select workspace output:\n%s", aTestOutput.All())
	}
	{
		ui := testUiWrapped(t)
		aView, aDone := testView(t)
		cApply := &ApplyCommand{
			Meta: Meta{
				Ui:                        ui,
				View:                      aView,
				AllowExperimentalFeatures: true,
			},
		}
		aCode := cApply.Run([]string{"-auto-approve"})
		aTestOutput := aDone(t)
		if aCode != 0 {
			t.Fatalf("bad: \n%s", aTestOutput.All())
		}

		t.Logf("Apply output:\n%s", aTestOutput.Stdout())
		t.Logf("Apply errors:\n%s", aTestOutput.Stderr())

		data := inmem.ReadState(t, "second")
		expectedOutputs := map[string]*states.OutputValue{
			"test": {
				Addr: addrs.AbsOutputValue{
					OutputValue: addrs.OutputValue{
						Name: "test",
					},
				},
				Value: cty.StringVal("test"),
			},
		}
		if diff := cmp.Diff(expectedOutputs, data.RootOutputValues); diff != "" {
			t.Fatalf("unexpected data after apply: %s", diff)
		}
	}
	{
		log.Printf("[TRACE] %s: beginning second init with state store", t.Name())

		ssCfg := `terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
  state_store "test_store" {
    provider "test" {}
    value = "foobar"
  }
}
`
		if err := os.WriteFile(filepath.Join(td, "main.tf"), []byte(ssCfg), 0644); err != nil {
			t.Fatalf("err: %s", err)
		}

		ui := testUiWrapped(t)
		view, done := testView(t)
		c := &InitCommand{
			Meta: Meta{
				testingOverrides:          tOverrides,
				ProviderSource:            providerSource,
				Ui:                        ui,
				View:                      view,
				AllowExperimentalFeatures: true,
			},
		}

		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
			"-force-copy",
			"-migrate-state",
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 1 {
			t.Fatalf("expected second init to exit with code 1, got %d:\n%s", code, testOutput.Stderr())
		}
		expectedErrMsgs := []string{
			"Error: Backend initialization required, please run \"terraform state migrate\" or \"terraform init -reconfigure\"",
			"Reason: Migrating from backend \"inmem\" to state store \"test_store\" in provider test (\"registry.terraform.io/hashicorp/test\")",
			"run \"terraform state migrate\"",
			"run \"terraform init -reconfigure\"",
		}
		output := cleanString(testOutput.Stderr())
		for _, expectedErr := range expectedErrMsgs {
			if !strings.Contains(output, expectedErr) {
				t.Fatalf("unexpected error, expected %q, given: %q", expectedErr, output)
			}
		}
	}

	// TODO: Create a test in state_migrate_test.go where the terraform state migrate command is used for the migration,
	// and assert both workspaces are migrated successfully.
}

func TestInit_cloud_to_stateStore(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()

	ts := cloud.TestServerWithHandlers(t, map[string]func(http.ResponseWriter, *http.Request){
		"/api/v2/organizations/hashicorp/workspaces/test": func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "GET" {
				w.Write([]byte(`{"data":{"id":"ws-TEST","type":"workspaces","attributes":{"allow-destroy-plan":true,"auto-apply":false,"auto-apply-run-trigger":false,"auto-destroy-activity-duration":null,"auto-destroy-at":null,"auto-destroy-status":null,"inherits-project-auto-destroy":true,"created-at":"2022-06-22T14:24:13.836Z","environment":"default","locked":false,"name":"test","queue-all-runs":false,"speculative-enabled":true,"structured-run-output-enabled":true,"terraform-version":"1.10.0","working-directory":null,"global-remote-state":false,"updated-at":"2026-01-29T15:09:18.075Z","resource-count":0,"apply-duration-average":2000,"plan-duration-average":4000,"policy-check-failures":0,"run-failures":0,"workspace-kpis-runs-count":1,"unarchived-workspace-change-requests-count":0,"latest-change-at":"2026-01-29T15:09:17.200Z","operations":true,"execution-mode":"remote","vcs-repo":null,"vcs-repo-identifier":null,"permissions":{"can-update":true,"can-destroy":true,"can-queue-run":true,"can-read-run":true,"can-read-variable":true,"can-update-variable":true,"can-read-state-versions":true,"can-read-state-outputs":true,"can-create-state-versions":true,"can-queue-apply":true,"can-lock":true,"can-unlock":true,"can-force-unlock":true,"can-read-settings":true,"can-manage-tags":true,"can-manage-run-tasks":true,"can-force-delete":true,"can-manage-assessments":true,"can-manage-ephemeral-workspaces":false,"can-read-assessment-results":true,"can-read-change-requests":false,"can-update-change-requests":false,"can-queue-destroy":true},"actions":{"is-destroyable":true},"description":null,"file-triggers-enabled":true,"trigger-prefixes":[],"trigger-patterns":[],"assessments-enabled":false,"last-assessment-result-at":null,"locked-reason":"","source":"terraform","source-name":null,"source-url":null,"tag-names":[],"setting-overwrites":{"execution-mode":true,"agent-pool":true}},"relationships":{"organization":{"data":{"id":"hashicorp","type":"organizations"}},"current-run":{"data":{"id":"run-TEST","type":"runs"},"links":{"related":"/api/v2/runs/run-TEST"}},"latest-run":{"data":{"id":"run-TEST","type":"runs"},"links":{"related":"/api/v2/runs/run-TEST"}},"outputs":{"data":[{"id":"wsout-TEST","type":"workspace-outputs"}],"links":{"related":"/api/v2/workspaces/ws-TEST/current-state-version-outputs"}},"remote-state-consumers":{"links":{"related":"/api/v2/workspaces/ws-TEST/relationships/remote-state-consumers"}},"current-state-version":{"data":{"id":"sv-TEST","type":"state-versions"},"links":{"related":"/api/v2/workspaces/ws-TEST/current-state-version"}},"current-configuration-version":{"data":{"id":"cv-TEST","type":"configuration-versions"},"links":{"related":"/api/v2/configuration-versions/cv-TEST"}},"agent-pool":{"data":null},"readme":{"data":null},"project":{"data":{"id":"prj-TEST","type":"projects"}},"current-assessment-result":{"data":null},"vars":{"data":[]}},"links":{"self":"/api/v2/organizations/hashicorp/workspaces/test","self-html":"/app/hashicorp/workspaces/test"}}}`))
				w.WriteHeader(http.StatusOK)
				return
			}
		},
		"/api/v2/workspaces/ws-TEST/current-state-version": func(w http.ResponseWriter, r *http.Request) {
			hostname := r.URL.Hostname()
			w.Write(fmt.Appendf([]byte{}, `{"data":{"id":"sv-TEST","type":"state-versions","attributes":{"created-at":"2026-01-29T15:09:17.200Z","size":651,"hosted-state-download-url":"%s/api/state-versions/sv-TEST/hosted_state","hosted-json-state-download-url":"%s/api/state-versions/sv-TEST/hosted_json_state","modules":{},"providers":{},"resources-processed":true,"serial":1,"state-version":4,"status":"finalized","terraform-version":"1.10.0","vcs-commit-url":null,"vcs-commit-sha":null,"resources":[],"billable-rum-count":0},"relationships":{"run":{"data":{"id":"run-TEST","type":"runs"}},"rollback-state-version":{"data":null},"created-by":{"data":{"id":"user-TEST","type":"users"},"links":{"self":"/api/v2/users/user-TEST","related":"/api/v2/runs/run-TEST/created-by"}},"workspace":{"data":{"id":"ws-TEST","type":"workspaces"}},"outputs":{"data":[{"id":"wsout-TEST","type":"state-version-outputs"}],"links":{"related":"/api/v2/state-versions/sv-TEST/outputs"}}},"links":{"self":"/api/v2/state-versions/sv-TEST"}}}`, hostname, hostname))
			w.WriteHeader(http.StatusOK)
		},
		"/api/state-versions/sv-TEST/hosted_state": func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"version":4,"terraform_version":"1.15.0","serial":1,"lineage":"91adaece-23b3-7bce-0695-5aea537d2fef","outputs":{"test":{"value":"test","type":"string"}},"resources":[],"check_results":null}`))
			w.WriteHeader(http.StatusOK)
		},
	})
	t.Cleanup(ts.Close)
	mockURL, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatal(err)
	}

	backendInit.Init(testDisco(ts))
	t.Cleanup(func() { backendInit.Init(nil) })

	cfg := fmt.Sprintf(`terraform {
  cloud {
    hostname = %q
    organization = "hashicorp"
    token = "test-token"
    workspaces {
      name = "test"
    }
  }
}
`, mockURL.Host)
	if err := os.WriteFile(filepath.Join(td, "main.tf"), []byte(cfg), 0644); err != nil {
		t.Fatalf("err: %s", err)
	}
	t.Chdir(td)

	mockProvider := mockPluggableStateStorageProvider(mockSingleStateStoreSchema("test_store"))
	mockProvider.MockStates = testing_provider.NewMockStateBytesWithStateIds("test_store", []string{"default"})
	mockProviderAddress := addrs.NewDefaultProvider("test")
	providerSource := newMockProviderSource(t, map[string][]string{
		"hashicorp/test": {"1.2.3"},
	})

	tOverrides := &testingOverrides{
		Providers: map[addrs.Provider]providers.Factory{
			mockProviderAddress: providers.FactoryFixed(mockProvider),
		},
	}

	{
		log.Printf("[TRACE] %s: beginning first init with backend", t.Name())
		// Init
		ui := testUiWrapped(t)
		view, done := testView(t)
		c := &InitCommand{
			Meta: Meta{
				Services:                  testDisco(ts),
				Ui:                        ui,
				View:                      view,
				AllowExperimentalFeatures: true,
			},
		}
		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 0 {
			t.Fatalf("bad: \n%s", testOutput.All())
		}
		log.Printf("[TRACE] %s: first init complete", t.Name())
		t.Logf("First run output:\n%s", testOutput.Stdout())
		t.Logf("First run errors:\n%s", testOutput.Stderr())
	}
	{
		log.Printf("[TRACE] %s: beginning second init with state store", t.Name())

		ssCfg := `terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
  state_store "test_store" {
    provider "test" {}
    value = "foobar"
  }
}
`
		if err := os.WriteFile(filepath.Join(td, "main.tf"), []byte(ssCfg), 0644); err != nil {
			t.Fatalf("err: %s", err)
		}

		ui := testUiWrapped(t)
		view, done := testView(t)
		c := &InitCommand{
			Meta: Meta{
				Services:                  testDisco(ts),
				testingOverrides:          tOverrides,
				ProviderSource:            providerSource,
				Ui:                        ui,
				View:                      view,
				AllowExperimentalFeatures: true,
			},
		}

		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
		}
		code := c.Run(args)
		testOutput := done(t)
		if code == 0 {
			t.Fatalf("expected migration from cloud to fail: \n%s", testOutput.Stdout())
		}
		log.Printf("[TRACE] %s: second init with state store complete", t.Name())
		expectedMsgs := []string{
			"Error: Backend initialization required, please run \"terraform state migrate\" or \"terraform init -reconfigure\"",
			"Reason: Migrating from backend \"cloud\" to state store \"test_store\" in provider test (\"registry.terraform.io/hashicorp/test\")",
		}
		output := cleanString(testOutput.Stderr())
		for _, expectedMsg := range expectedMsgs {
			if !strings.Contains(output, expectedMsg) {
				t.Fatalf("expected error message %q not found: \n%s", expectedMsg, output)
			}
		}
	}
}

// Test that config-parsing errors that prevent initialising the pluggable state store are identified and returned
// before Terraform attempts to initialise the store.
//
// These errors include omitting the necessary entry in required_providers, or causing an issue with how require_providers
// is parsed. This test uses the first scenario for simplicity.
func TestInit_configErrorsImpactingStateStore(t *testing.T) {
	td := t.TempDir()
	t.Chdir(td)
	cfg1 := `terraform {
  required_providers {
    foobar = {
      source = "hashicorp/foobar"
    }
  }
  state_store "test_store" {
    provider "test" {} # missing from required_providers
    value = "foobar"
  }
}
	`
	if err := os.WriteFile(filepath.Join(td, "main.tf"), []byte(cfg1), 0644); err != nil {
		t.Fatalf("err: %s", err)
	}

	ui := testUiWrapped(t)
	view, done := testView(t)
	initCmd := &InitCommand{
		Meta: Meta{
			Ui:                        ui,
			View:                      view,
			AllowExperimentalFeatures: true,
		},
	}

	log.Printf("[TRACE] TestInit_configErrorsImpactingStateStore: init start")
	args := []string{"-enable-pluggable-state-storage-experiment"}
	code := initCmd.Run(args)
	testOutput := done(t)
	if code != 1 {
		t.Fatalf("expected init to fail with code 1, got code %d: \n%s", code, testOutput.All())
	}
	log.Printf("[TRACE] TestInit_configErrorsImpactingStateStore: init complete")
	t.Logf("init output:\n%s", testOutput.Stdout())
	t.Logf("init errors:\n%s", testOutput.Stderr())

	expectedErrs := []string{
		// Pre-amble text that's shown when a config-parsing error occurs during init.
		"Error: Terraform encountered problems during initialisation, including problems with the configuration, described below.",
		// This parsing error previously wouldn't be reported before initialising the backend, so
		// Terraform attempted to use a state store in the missing provider.
		"Error: Missing entry in required_providers",
	}
	for _, e := range expectedErrs {
		if !strings.Contains(cleanString(testOutput.Stderr()), e) {
			t.Fatalf("unexpected error, expected %q, given: %s", e, testOutput.Stderr())
		}
	}
}
