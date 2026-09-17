// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"fmt"
	"os"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

// validatePolicyPaths checks that each of the given policy paths refers to
// something that exists and is readable. Both files and directories are
// accepted here, which is the behavior expected by "terraform init", "plan",
// and "apply".
func validatePolicyPaths(policyPaths []string) (diags tfdiags.Diagnostics) {
	for _, path := range policyPaths {
		if _, err := os.Stat(path); err != nil {
			if os.IsNotExist(err) {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Invalid policy path",
					fmt.Sprintf("Terraform cannot find the policy path at %s. Please ensure the file or directory exists and the path is correct.", path),
				))
				continue
			}

			diags = diags.Append(policyPathReadErrorDiagnostic(path, err))
		}
	}
	return diags
}

// validatePolicySetDirs checks that each of the given policy paths refers to an
// existing directory. Commands whose -policies option is documented as taking a
// policy set directory use this instead of validatePolicyPaths, so that a path
// to a single file is rejected with an actionable diagnostic rather than being
// passed along to the policy engine.
//
// Symlinks are resolved before the check, so a symlink to a directory is
// accepted while a symlink to a file or a dangling symlink is not.
func validatePolicySetDirs(policyPaths []string) (diags tfdiags.Diagnostics) {
	for _, path := range policyPaths {
		// An empty value, such as from "-policies=", has no path to report
		// back to the user, so it gets its own message.
		if path == "" {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Invalid policy path",
				"The -policies option requires the path of a policy set directory, but was given an empty value.",
			))
			continue
		}

		info, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Invalid policy path",
					fmt.Sprintf("Terraform cannot find the policy path at %s. The -policies option requires the path of an existing policy set directory.", path),
				))
				continue
			}

			diags = diags.Append(policyPathReadErrorDiagnostic(path, err))
			continue
		}

		if !info.IsDir() {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Invalid policy path",
				fmt.Sprintf("The policy path %s is not a directory. The -policies option requires the path of a policy set directory, not an individual policy file.", path),
			))
		}
	}
	return diags
}

// policyPathReadErrorDiagnostic builds the diagnostic for a policy path that
// could not be inspected on disk for a reason other than it not existing.
func policyPathReadErrorDiagnostic(path string, err error) tfdiags.Diagnostic {
	return tfdiags.Sourceless(
		tfdiags.Error,
		"Invalid policy path",
		fmt.Sprintf("Terraform could not read the policy path at %s: %s.", path, err),
	)
}
