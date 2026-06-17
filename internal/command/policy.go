// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

func validatePolicyPaths(policyPaths []string, experimental bool) (diags tfdiags.Diagnostics) {
	if !experimental && len(policyPaths) > 0 {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to parse command-line flags",
			"The -policies flag is only valid in experimental builds of Terraform.",
		))
	}

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

			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Invalid policy path",
				fmt.Sprintf("Terraform could not read the policy path at %s: %s.", path, err),
			))
		}
	}
	return diags
}

// validatePolicyContentPaths checks that each path in policyPaths points to an
// existing directory containing at least one policy file. A policy file is any
// file ending in .policy.hcl. The experimental gate is also enforced so this
// function can serve as the sole validator for commands that require
// directory-level policy validation.
func validatePolicyContentPaths(policyPaths []string, experimental bool) (diags tfdiags.Diagnostics) {
	if !experimental && len(policyPaths) > 0 {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to parse command-line flags",
			"The -policies flag is only valid in experimental builds of Terraform.",
		))
	}

	for _, path := range policyPaths {
		info, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Invalid policy path",
					fmt.Sprintf("Terraform cannot find the policy path at %s. Please ensure the file or directory exists and the path is correct.", path),
				))
				continue
			}
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Invalid policy path",
				fmt.Sprintf("Terraform could not read the policy path at %s: %s.", path, err),
			))
			continue
		}

		if !info.IsDir() {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Invalid policy path",
				fmt.Sprintf("The policy path %s is not a directory. Each -policies path must point to a directory containing policy files.", path),
			))
			continue
		}

		entries, err := os.ReadDir(path)
		if err != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Invalid policy path",
				fmt.Sprintf("Terraform could not read the policy directory at %s: %s.", path, err),
			))
			continue
		}

		hasPolicy := false
		for _, e := range entries {
			if !e.IsDir() && (strings.HasSuffix(e.Name(), ".policy.hcl")) {
				hasPolicy = true
				break
			}
		}

		if !hasPolicy {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Invalid policy path",
				fmt.Sprintf("The policy directory at %s contains no policy files. Ensure the directory contains at least one file ending in .policy.hcl.", path),
			))
		}
	}
	return diags
}
