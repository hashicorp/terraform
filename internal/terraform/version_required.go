// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package terraform

import (
	"github.com/hashicorp/terraform/internal/tfdiags"

	"github.com/hashicorp/terraform/internal/configs"
)

// CheckCoreVersionRequirements visits each of the modules in the given
// configuration tree and verifies that any given Core version constraints
// match with the version of Terraform Core that is being used.
//
// The returned diagnostics will contain errors if any constraints do not match.
// The returned diagnostics might also return warnings, which should be
// displayed to the user.
func CheckCoreVersionRequirements(config *configs.Config) tfdiags.Diagnostics {
	if config == nil {
		return nil
	}

	var diags tfdiags.Diagnostics
	diags = diags.Append(config.CheckCoreVersionRequirements())

	return diags
}
