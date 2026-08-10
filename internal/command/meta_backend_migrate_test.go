// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"strings"
	"testing"

	backendLocal "github.com/hashicorp/terraform/internal/backend/local"
	"github.com/hashicorp/terraform/internal/backend/pluggable"
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
