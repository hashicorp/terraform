// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"fmt"
	"os"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

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

			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Invalid policy path",
				fmt.Sprintf("Terraform could not read the policy path at %s: %s.", path, err),
			))
		}
	}
	return diags
}
