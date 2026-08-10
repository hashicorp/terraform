// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	backendLocal "github.com/hashicorp/terraform/internal/backend/local"
	"github.com/hashicorp/terraform/internal/backend/pluggable"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/zclconf/go-cty/cty"
)

func Test_backendMigrateState_S_S(t *testing.T) {
	storeType := "test_store"
	p := mockPluggableStateStorageProvider(mockSingleStateStoreSchema(storeType))
	source, err := pluggable.NewPluggable(p, storeType)
	if err != nil {
		t.Fatalf("unexpected err: %s", err)
	}
	destination := backendLocal.New() // local

	opts := &backendMigrateOpts{
		SourceType: storeType,
		Source:     source,

		DestinationType: "local",
		Destination:     destination,
	}

	inputWriter := testInputMap(t, map[string]string{
		"backend-migrate-multistate-to-multistate": "no", // We're only testing the prompt, no is sufficient
	})

	meta := Meta{
		input: true,
		Ui:    testUiWrapped(t),
	}
	err = meta.backendMigrateState_S_S(opts)
	if err == nil {
		t.Fatal("expected a 'Migration aborted by user' error but got none")
	}

	// Check the source and destination are interpolated in the expected places
	expected := `the existing "test_store" state store and the newly configured "local" backend`
	if !strings.Contains(inputWriter.String(), expected) {
		t.Fatalf("expected the input prompt to include %q, but got':\n %s", expected, inputWriter.String())
	}
}

func Test_backendMigrateState_S_s(t *testing.T) {
	storeType := "test_store"
	p := mockPluggableStateStorageProvider(mockSingleStateStoreSchema(storeType))
	p.ConfigureProviderCalled = true
	p.ConfigureStateStoreCalled = true
	source, err := pluggable.NewPluggable(p, storeType)
	if err != nil {
		t.Fatalf("unexpected err: %s", err)
	}
	destination := backendLocal.TestNewLocalSingle() // local with no workspace support

	opts := &backendMigrateOpts{
		SourceType: storeType,
		Source:     source,

		DestinationType: "local",
		Destination:     destination,
	}

	inputWriter := testInputMap(t, map[string]string{
		"backend-migrate-multistate-to-single": "no", // We're only testing the prompt, no is sufficient
	})

	meta := Meta{
		input: true,
		Ui:    testUiWrapped(t),
	}
	err = meta.backendMigrateState_S_s(opts)
	if err == nil {
		t.Fatal("expected a 'Migration aborted by user' error but got none")
	}

	// Check the source and destination are interpolated in the expected places
	expected := []string{
		`Destination backend "local" doesn't support workspaces`,
		`The existing "test_store" state store supports workspaces`,
	}
	for _, e := range expected {
		if !strings.Contains(inputWriter.String(), e) {
			t.Fatalf("expected the input prompt to include %q, but got':\n %s", e, inputWriter.String())
		}
	}
}

func Test_backendMigrateState_s_s(t *testing.T) {
	t.Run("no state in destination - prompt via backendMigrateEmptyConfirm", func(t *testing.T) {
		workspaceName := "default"

		storeType := "test_store"
		p := mockPluggableStateStorageProvider(mockSingleStateStoreSchema(storeType))
		p.ConfigureProviderCalled = true
		p.ConfigureStateStoreCalled = true

		// We need a state to migrate
		p.MockStates = testing_provider.NewMockStateBytesWithSingleState(
			"test_store",
			workspaceName,
			[]byte(`{"version":4,"terraform_version":"1.16.0","serial":1,"lineage":"8595356e-c764-dea1-8dae-156b936ec6e2","outputs":{"test":{"value":"test","type":"string"}},"resources":[],"check_results":null}`),
		)
		source, err := pluggable.NewPluggable(p, storeType)
		if err != nil {
			t.Fatalf("unexpected err: %s", err)
		}
		destination := backendLocal.New()

		opts := &backendMigrateOpts{
			SourceType: storeType,
			Source:     source,

			DestinationType: "local",
			Destination:     destination,

			sourceWorkspace: workspaceName,
		}

		inputWriter := testInputMap(t, map[string]string{
			"backend-migrate-copy-to-empty": "no", // We're only testing the prompt, no is sufficient
		})

		meta := Meta{
			input: true,
			Ui:    testUiWrapped(t),
		}
		err = meta.backendMigrateState_s_s(opts)
		if err != nil {
			t.Fatalf("Didn't expect an error but got: %s", err)
		}

		// Check the source and destination are interpolated in the expected places
		promptText := cleanString(inputWriter.String()) // assertions need to space newlines, this makes it easier
		expected := []string{
			`migrating the previous "test_store" state store to the newly configured "local" backend.`,
			`No existing state was found in the newly configured "local" backend`,
			`copy this state to the new "local" backend?`,
		}
		for _, e := range expected {
			if !strings.Contains(promptText, e) {
				t.Fatalf("expected the input prompt to include %q, but got':\n %s", e, promptText)
			}
		}
	})

	t.Run("state in destination - prompt via backendMigrateNonEmptyConfirm", func(t *testing.T) {
		td := t.TempDir()
		t.Chdir(td)

		workspaceName := "default"

		// Source - has a state for migration
		storeType := "test_store"
		p := mockPluggableStateStorageProvider(mockSingleStateStoreSchema(storeType))
		p.ConfigureProviderCalled = true
		p.ConfigureStateStoreCalled = true
		p.MockStates = testing_provider.NewMockStateBytesWithSingleState(
			"test_store",
			workspaceName,
			[]byte(`{"version":4,"terraform_version":"1.16.0","serial":1,"lineage":"8595356e-c764-dea1-8dae-156b936ec6e2","outputs":{"test":{"value":"test","type":"string"}},"resources":[],"check_results":null}`),
		)
		source, err := pluggable.NewPluggable(p, storeType)
		if err != nil {
			t.Fatalf("unexpected err: %s", err)
		}

		// Destination - has a state present to cause the prompt under test
		state := states.NewState()
		state.SetOutputValue(
			addrs.OutputValue{Name: "bar"}.Absolute(addrs.RootModuleInstance),
			cty.StringVal("bar value"), false,
		)
		state.SetOutputValue(
			addrs.OutputValue{Name: "secret"}.Absolute(addrs.RootModuleInstance),
			cty.StringVal("secret value"), true,
		)
		destination := backendLocal.New()
		dStateMgr, diags := destination.StateMgr(workspaceName)
		if diags.HasErrors() {
			t.Fatalf("unexpected error: %s", err)
		}
		err = dStateMgr.WriteState(state)
		if err != nil {
			t.Fatalf("unexpected err: %s", err)
		}

		opts := &backendMigrateOpts{
			SourceType: storeType,
			Source:     source,

			DestinationType: "local",
			Destination:     destination,

			sourceWorkspace: workspaceName,
		}

		inputWriter := testInputMap(t, map[string]string{
			"backend-migrate-to-backend": "no", // We're only testing the prompt, no is sufficient
		})

		meta := Meta{
			input: true,
			Ui:    testUiWrapped(t),
		}
		err = meta.backendMigrateState_s_s(opts)
		if err != nil {
			t.Fatalf("Didn't expect an error but got: %s", err)
		}

		// Check the source and destination are interpolated in the expected places
		promptText := cleanString(inputWriter.String()) // assertions need to space newlines, this makes it easier
		expected := []string{
			`migrating the previous "test_store" state store to the newly configured "local" backend.`,
			`non-empty state already exists in the new backend.`,
			`Previous (state store "test_store"):`,
			`New (backend "local"):`,
			`overwrite the state in the new backend with the previous state?`,
		}
		for _, e := range expected {
			if !strings.Contains(promptText, e) {
				t.Fatalf("expected the input prompt to include %q, but got':\n %s", e, promptText)
			}
		}
	})
}

func TestBackendMigrate_promptMultiStatePattern(t *testing.T) {
	// Setup the meta

	cases := map[string]struct {
		renamePrompt  string
		patternPrompt string
		expectedErr   string
	}{
		"valid pattern": {
			renamePrompt:  "1",
			patternPrompt: "hello-*",
			expectedErr:   "",
		},
		"invalid pattern, only one asterisk allowed": {
			renamePrompt:  "1",
			patternPrompt: "hello-*-world-*",
			expectedErr:   "The pattern '*' cannot be used more than once.",
		},
		"invalid pattern, missing asterisk": {
			renamePrompt:  "1",
			patternPrompt: "hello-world",
			expectedErr:   "The pattern must have an '*'",
		},
		"invalid rename": {
			renamePrompt: "3",
			expectedErr:  "Please select 1 or 2 as part of this option.",
		},
		"no rename": {
			renamePrompt: "2",
		},
	}
	for name, tc := range cases {
		t.Log("Test: ", name)
		m := testMetaBackend(t, nil)
		input := map[string]string{}
		inputWriter := testInputMap(t, input)
		if tc.renamePrompt != "" {
			input["backend-migrate-multistate-to-tfc"] = tc.renamePrompt
		}
		if tc.patternPrompt != "" {
			input["backend-migrate-multistate-to-tfc-pattern"] = tc.patternPrompt
		}

		sourceType := "s3"
		_, err := m.promptMultiStateMigrationPattern(sourceType, "backend", "HCP Terraform")
		if tc.expectedErr == "" && err != nil {
			t.Fatalf("expected error to be nil, but was %s", err.Error())
		}
		if tc.expectedErr != "" && tc.expectedErr != err.Error() {
			t.Fatalf("expected error to eq %s but got %s", tc.expectedErr, err.Error())
		}

		// Check prompt text uses arguments in expected way
		expected := `migrating existing workspaces from the backend "s3" to HCP Terraform`
		if !strings.Contains(inputWriter.String(), expected) {
			t.Fatalf("expected the input prompt to include: %s\nbut got:\n %s", expected, inputWriter.String())
		}
	}
}
