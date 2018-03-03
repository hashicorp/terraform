package terraform

import (
	"fmt"

	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/tfdiags"
	tfversion "github.com/hashicorp/terraform/version"
)

// CheckRequiredVersion verifies that any version requirements specified by
// the configuration are met.
//
// This checks the root module as well as any additional version requirements
// from child modules.
//
// This is tested in context_test.go.
func CheckRequiredVersion(cfg *configs.Config) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	for _, constraint := range cfg.Module.CoreVersionConstraints {
		if !constraint.Required.Check(tfversion.SemVer) {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Unmet Terraform version requirement",
				Detail:   fmt.Sprintf(checkRequiredVersionDetailFormat, tfversion.SemVer),
				Subject:  &constraint.DeclRange,
			})
		}
	}

	for _, child := range cfg.Children {
		childDiags := CheckRequiredVersion(child)
		diags = diags.Append(childDiags)
	}

	return diags
}

const checkRequiredVersionDetailFormat = `Your current Terraform Core version %s does not meet this version constraint.

To proceed, either switch to an allowed version or update the configuration to permit your current version.

Version requirements are usually set for a good reason, so check with whoever set this version constraint before adjusting it.`
