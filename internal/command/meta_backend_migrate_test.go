// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

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
		cleanup := testInputMap(t, input)
		if tc.renamePrompt != "" {
			input["backend-migrate-multistate-to-tfc"] = tc.renamePrompt
		}
		if tc.patternPrompt != "" {
			input["backend-migrate-multistate-to-tfc-pattern"] = tc.patternPrompt
		}

		sourceType := "cloud"
		_, err := m.promptMultiStateMigrationPattern(sourceType)
		if tc.expectedErr == "" && err != nil {
			t.Fatalf("expected error to be nil, but was %s", err.Error())
		}
		if tc.expectedErr != "" && tc.expectedErr != err.Error() {
			t.Fatalf("expected error to eq %s but got %s", tc.expectedErr, err.Error())
		}

		cleanup()
	}
}
