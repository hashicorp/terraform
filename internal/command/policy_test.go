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

func TestValidatePolicyPathsContent(t *testing.T) {
	validDirA := t.TempDir()
	if err := os.WriteFile(filepath.Join(validDirA, "main.policy.hcl"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	validDirB := t.TempDir()
	if err := os.WriteFile(filepath.Join(validDirB, "allow.policy.hcl"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	validDirC := t.TempDir()
	if err := os.WriteFile(filepath.Join(validDirC, ".policy.hcl"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	fileNotDir := filepath.Join(t.TempDir(), "policy.hcl")
	if err := os.WriteFile(fileNotDir, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	emptyDir := t.TempDir()
	missingPath := filepath.Join(t.TempDir(), "does-not-exist")

	tests := []struct {
		name         string
		policyPaths  []string
		experimental bool
		wantDiags    tfdiags.Diagnostics
	}{
		{
			name:         "empty slice",
			policyPaths:  nil,
			experimental: true,
		},
		{
			name:         "valid directory with {prefix}.policy.hcl",
			policyPaths:  []string{validDirA},
			experimental: true,
		},
		{
			name:         "valid directory with .policy.hcl file",
			policyPaths:  []string{validDirB},
			experimental: true,
		},
		{
			name:         "valid directory with .policy.hcl (no prefix)",
			policyPaths:  []string{validDirC},
			experimental: true,
		},
		{
			name:         "path is a file not a directory",
			policyPaths:  []string{fileNotDir},
			experimental: true,
			wantDiags: tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Invalid policy path",
					fmt.Sprintf("The policy path %s is not a directory. Each -policies path must point to a directory containing policy files.", fileNotDir),
				),
			},
		},
		{
			name:         "directory with no policy files",
			policyPaths:  []string{emptyDir},
			experimental: true,
			wantDiags: tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Invalid policy path",
					fmt.Sprintf("The policy directory at %s contains no policy files. Ensure the directory contains at least one file ending in .policy.hcl.", emptyDir),
				),
			},
		},
		{
			name:         "non-existent path",
			policyPaths:  []string{missingPath},
			experimental: true,
			wantDiags: tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Invalid policy path",
					fmt.Sprintf("Terraform cannot find the policy path at %s. Please ensure the file or directory exists and the path is correct.", missingPath),
				),
			},
		},
		{
			name:         "experiments disallowed",
			policyPaths:  []string{validDirA},
			experimental: false,
			wantDiags: tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to parse command-line flags",
					"The -policies flag is only valid in experimental builds of Terraform.",
				),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := validatePolicyContentPaths(tc.policyPaths, tc.experimental)
			if tc.wantDiags == nil {
				tfdiags.AssertNoDiagnostics(t, got)
				return
			}
			tfdiags.AssertDiagnosticsMatch(t, got, tc.wantDiags)
		})
	}
}
