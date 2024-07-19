// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package jsonformat

import (
	"testing"

	"github.com/hashicorp/terraform/internal/command/jsonplan"
	"github.com/hashicorp/terraform/internal/command/jsonprovider"
	"github.com/hashicorp/terraform/internal/command/jsonstate"
)

func TestIncompatibleVersions(t *testing.T) {
	tcs := map[string]struct {
		local    string
		remote   string
		expected bool
	}{
		"matching": {
			local:    "1.1",
			remote:   "1.1",
			expected: false,
		},
		"local_latest": {
			local:    "1.2",
			remote:   "1.1",
			expected: false,
		},
		"local_earliest": {
			local:    "1.1",
			remote:   "1.2",
			expected: true,
		},
		"parses_state_version": {
			local:    jsonstate.FormatVersion,
			remote:   jsonstate.FormatVersion,
			expected: false,
		},
		"parses_provider_version": {
			local:    jsonprovider.FormatVersion,
			remote:   jsonprovider.FormatVersion,
			expected: false,
		},
		"parses_plan_version": {
			local:    jsonplan.FormatVersion,
			remote:   jsonplan.FormatVersion,
			expected: false,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			actual := incompatibleVersions(tc.local, tc.remote)
			if actual != tc.expected {
				t.Errorf("expected %t but found %t", tc.expected, actual)
			}
		})
	}
}
