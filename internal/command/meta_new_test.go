// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"testing"

	"github.com/hashicorp/terraform/internal/command/ui"
)

func TestMeta_Input(t *testing.T) {
	cases := []struct {
		name     string
		meta     *Meta
		expected bool
		env      string
	}{
		{
			name: "E2E usage with -input=true => input enabled",
			// This is exactly the same as a badly set up test,
			// which would fail due to no input being provided.
			meta: &Meta{
				input:            true, // default, CLI flag redundant
				testingOverrides: nil,  // unset as not in a test
			},
			expected: true,
		},
		{
			name: "E2E usage with -input=false => input disabled",
			meta: &Meta{
				input:            false, // set by CLI flag
				testingOverrides: nil,   // unset as not in a test
			},
			expected: false,
		},
		{
			name: "E2E usage with TF_INPUT=false => input disabled",
			meta: &Meta{
				input:            false, // changed by env
				testingOverrides: nil,   // unset as not in a test
			},
			env:      "false",
			expected: false,
		},
		{
			name: "E2E usage with TF_INPUT=0 => input disabled",
			meta: &Meta{
				input:            false, // changed by env
				testingOverrides: nil,   // unset as not in a test
			},
			env:      "0",
			expected: false,
		},
		{
			name: "test usage with UIInput not set in testingOverrides => input disabled",
			meta: &Meta{
				input: true, // default
				testingOverrides: &testingOverrides{
					UIInput: nil, // unset in test setup
				},
			},
			expected: false,
		},
		{
			name: "test usage with UIInput set in testingOverrides => input enabled",
			meta: &Meta{
				input: true, // default
				testingOverrides: &testingOverrides{
					UIInput: ui.NewUIInputForTests(ui.UIInputOptions{}, nil, nil),
				},
			},
			expected: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.env != "" {
				t.Setenv(InputModeEnvVar, tc.env)
			}

			got := tc.meta.Input()
			if got != tc.expected {
				t.Errorf("got %v; want %v", got, tc.expected)
			}
		})
	}
}
