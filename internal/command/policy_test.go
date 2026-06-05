// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestValidatePolicyPaths(t *testing.T) {
	td := t.TempDir()
	existingPath := filepath.Join(td, "policy.tfpolicy.hcl")
	if err := os.WriteFile(existingPath, []byte("resource_policy \"test_resource\" \"allow\" {}\n"), os.FileMode(os.O_RDWR)); err != nil {
		t.Fatal(err)
	}
	missingPath := filepath.Join(td, "missing")

	tests := []struct {
		name             string
		path             string
		want             tfdiags.Diagnostics
		allowExperiments bool
	}{
		{
			name:             "existing path",
			path:             existingPath,
			allowExperiments: true,
		},
		{
			name:             "missing path",
			path:             missingPath,
			allowExperiments: true,
			want: tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Invalid policy path",
					fmt.Sprintf("Terraform cannot find the policy path at %s. Please ensure the file or directory exists and the path is correct.", missingPath),
				),
			},
		},
		{
			name:             "existing path, experiments disallowed",
			path:             existingPath,
			allowExperiments: false,
			want: tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to parse command-line flags",
					"The -policies flag is only valid in experimental builds of Terraform.",
				),
			},
		},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("plan: %s", tc.name), func(t *testing.T) {
			cmd := &PlanCommand{Meta: Meta{AllowExperimentalFeatures: tc.allowExperiments}}
			got := cmd.Validate(&arguments.Plan{PolicyPaths: []string{tc.path}})
			if tc.want == nil {
				tfdiags.AssertNoDiagnostics(t, got)
				return
			}
			tfdiags.AssertDiagnosticsMatch(t, got, tc.want)
		})
		t.Run(fmt.Sprintf("apply: %s", tc.name), func(t *testing.T) {
			cmd := &ApplyCommand{Meta: Meta{AllowExperimentalFeatures: tc.allowExperiments}}
			got := cmd.Validate(&arguments.Apply{PolicyPaths: []string{tc.path}})
			if tc.want == nil {
				tfdiags.AssertNoDiagnostics(t, got)
				return
			}
			tfdiags.AssertDiagnosticsMatch(t, got, tc.want)
		})
	}
}
