package terraform

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/tfdiags"

	"github.com/hashicorp/terraform/configs"

	tfversion "github.com/hashicorp/terraform/version"
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
	module := config.Module

	for _, constraint := range module.CoreVersionConstraints {
		if !constraint.Required.Check(tfversion.SemVer) {
			switch {
			case len(config.Path) == 0:
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Unsupported Terraform Core version",
					Detail: fmt.Sprintf(
						"This configuration does not support Terraform version %s. To proceed, either choose another supported Terraform version or update this version constraint. Version constraints are normally set for good reason, so updating the constraint may lead to other errors or unexpected behavior.",
						tfversion.String(),
					),
					Subject: &constraint.DeclRange,
				})
			default:
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Unsupported Terraform Core version",
					Detail: fmt.Sprintf(
						"Module %s (from %s) does not support Terraform version %s. To proceed, either choose another supported Terraform version or update this version constraint. Version constraints are normally set for good reason, so updating the constraint may lead to other errors or unexpected behavior.",
						config.Path, config.SourceAddr, tfversion.String(),
					),
					Subject: &constraint.DeclRange,
				})
			}
		}
	}

	for _, c := range config.Children {
		childDiags := CheckCoreVersionRequirements(c)
		diags = diags.Append(childDiags)
	}

	return diags
}
