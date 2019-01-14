package initwd

import (
	"fmt"

	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform/internal/earlyconfig"
	"github.com/hashicorp/terraform/tfdiags"
	tfversion "github.com/hashicorp/terraform/version"
)

// CheckCoreVersionRequirements visits each of the modules in the given
// early configuration tree and verifies that any given Core version constraints
// match with the version of Terraform Core that is being used.
//
// The returned diagnostics will contain errors if any constraints do not match.
// The returned diagnostics might also return warnings, which should be
// displayed to the user.
func CheckCoreVersionRequirements(earlyConfig *earlyconfig.Config) tfdiags.Diagnostics {
	if earlyConfig == nil {
		return nil
	}

	var diags tfdiags.Diagnostics
	module := earlyConfig.Module

	var constraints version.Constraints
	for _, constraintStr := range module.RequiredCore {
		constraint, err := version.NewConstraint(constraintStr)
		if err != nil {
			// Unfortunately the early config parser doesn't preserve a source
			// location for this, so we're unable to indicate a specific
			// location where this constraint came from, but we can at least
			// say which module set it.
			switch {
			case len(earlyConfig.Path) == 0:
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Invalid provider version constraint",
					fmt.Sprintf("Invalid version core constraint %q in the root module.", constraintStr),
				))
			default:
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Invalid provider version constraint",
					fmt.Sprintf("Invalid version core constraint %q in %s.", constraintStr, earlyConfig.Path),
				))
			}
			continue
		}
		constraints = append(constraints, constraint...)
	}

	if !constraints.Check(tfversion.SemVer) {
		switch {
		case len(earlyConfig.Path) == 0:
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Unsupported Terraform Core version",
				fmt.Sprintf(
					"This configuration does not support Terraform version %s. To proceed, either choose another supported Terraform version or update the root module's version constraint. Version constraints are normally set for good reason, so updating the constraint may lead to other errors or unexpected behavior.",
					tfversion.String(),
				),
			))
		default:
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Unsupported Terraform Core version",
				fmt.Sprintf(
					"Module %s (from %q) does not support Terraform version %s. To proceed, either choose another supported Terraform version or update the module's version constraint. Version constraints are normally set for good reason, so updating the constraint may lead to other errors or unexpected behavior.",
					earlyConfig.Path, earlyConfig.SourceAddr, tfversion.String(),
				),
			))
		}
	}

	for _, c := range earlyConfig.Children {
		childDiags := CheckCoreVersionRequirements(c)
		diags = diags.Append(childDiags)
	}

	return diags
}
