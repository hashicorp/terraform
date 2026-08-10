// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"testing"
)

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
