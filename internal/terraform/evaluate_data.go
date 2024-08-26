// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/didyoumean"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// evaluationData is the base struct for evaluating data from within Terraform
// Core. It contains some common data and functions shared by the various
// implemented evaluators.
type evaluationData struct {
	Evaluator *Evaluator

	// Module is the unexpanded module that this data is being evaluated within.
	Module addrs.Module
}

// GetPathAttr implements lang.Data.
func (d *evaluationData) GetPathAttr(addr addrs.PathAttr, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	switch addr.Name {

	case "cwd":
		var err error
		var wd string
		if d.Evaluator.Meta != nil {
			// Meta is always non-nil in the normal case, but some test cases
			// are not so realistic.
			wd = d.Evaluator.Meta.OriginalWorkingDir
		}
		if wd == "" {
			wd, err = os.Getwd()
			if err != nil {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  `Failed to get working directory`,
					Detail:   fmt.Sprintf(`The value for path.cwd cannot be determined due to a system error: %s`, err),
					Subject:  rng.ToHCL().Ptr(),
				})
				return cty.DynamicVal, diags
			}
		}
		// The current working directory should always be absolute, whether we
		// just looked it up or whether we were relying on ContextMeta's
		// (possibly non-normalized) path.
		wd, err = filepath.Abs(wd)
		if err != nil {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  `Failed to get working directory`,
				Detail:   fmt.Sprintf(`The value for path.cwd cannot be determined due to a system error: %s`, err),
				Subject:  rng.ToHCL().Ptr(),
			})
			return cty.DynamicVal, diags
		}

		return cty.StringVal(filepath.ToSlash(wd)), diags

	case "module":
		moduleConfig := d.Evaluator.Config.Descendent(d.Module)
		if moduleConfig == nil {
			// should never happen, since we can't be evaluating in a module
			// that wasn't mentioned in configuration.
			panic(fmt.Sprintf("module.path read from module %s, which has no configuration", d.Module))
		}
		sourceDir := moduleConfig.Module.SourceDir
		return cty.StringVal(filepath.ToSlash(sourceDir)), diags

	case "root":
		sourceDir := d.Evaluator.Config.Module.SourceDir
		return cty.StringVal(filepath.ToSlash(sourceDir)), diags

	default:
		suggestion := didyoumean.NameSuggestion(addr.Name, []string{"cwd", "module", "root"})
		if suggestion != "" {
			suggestion = fmt.Sprintf(" Did you mean %q?", suggestion)
		}
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  `Invalid "path" attribute`,
			Detail:   fmt.Sprintf(`The "path" object does not have an attribute named %q.%s`, addr.Name, suggestion),
			Subject:  rng.ToHCL().Ptr(),
		})
		return cty.DynamicVal, diags
	}
}

// GetTerraformAttr implements lang.Data.
func (d *evaluationData) GetTerraformAttr(addr addrs.TerraformAttr, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	switch addr.Name {

	case "workspace":
		// The absence of an "env" (really: workspace) name suggests that
		// we're running in a non-workspace context, such as in a component
		// of a stack. terraform.workspace is a legacy thing from workspaces
		// mode that isn't carried forward to stacks, because stack
		// configurations can instead vary their behavior based on input
		// variables provided in the deployment configuration.
		if d.Evaluator.Meta == nil || d.Evaluator.Meta.Env == "" {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  `Invalid reference`,
				Detail:   `The terraform.workspace attribute is only available for modules used in Terraform workspaces. Use input variables instead to create variations between different instances of this module.`,
				Subject:  rng.ToHCL().Ptr(),
			})
			return cty.DynamicVal, diags
		}
		workspaceName := d.Evaluator.Meta.Env
		return cty.StringVal(workspaceName), diags

	// terraform.applying is an ephemeral boolean value that's set to true
	// during an apply walk or false in any other situation. This is
	// intended to allow, for example, using a more privileged auth role
	// in a provider configuration during the apply phase but a more
	// constrained role for other situations.
	case "applying":
		return cty.BoolVal(d.Evaluator.Operation == walkApply).Mark(marks.Ephemeral), nil

	case "env":
		// Prior to Terraform 0.12 there was an attribute "env", which was
		// an alias name for "workspace". This was deprecated and is now
		// removed.
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  `Invalid "terraform" attribute`,
			Detail:   `The terraform.env attribute was deprecated in v0.10 and removed in v0.12. The "state environment" concept was renamed to "workspace" in v0.12, and so the workspace name can now be accessed using the terraform.workspace attribute.`,
			Subject:  rng.ToHCL().Ptr(),
		})
		return cty.DynamicVal, diags

	default:
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  `Invalid "terraform" attribute`,
			Detail:   fmt.Sprintf(`The "terraform" object does not have an attribute named %q. The only supported attributes are terraform.workspace, the name of the currently-selected workspace, and terraform.applying, a boolean which is true only during apply.`, addr.Name),
			Subject:  rng.ToHCL().Ptr(),
		})
		return cty.DynamicVal, diags
	}
}

// StaticValidateReferences implements lang.Data.
func (d *evaluationData) StaticValidateReferences(refs []*addrs.Reference, self addrs.Referenceable, source addrs.Referenceable) tfdiags.Diagnostics {
	return d.Evaluator.StaticValidateReferences(refs, d.Module, self, source)
}

// GetRunBlock implements lang.Data.
func (d *evaluationData) GetRunBlock(addrs.Run, tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	// We should not get here because any scope that has an [evaluationPlaceholderData]
	// as its Data should have a reference parser that doesn't accept addrs.Run
	// addresses.
	panic("GetRunBlock called on non-test evaluation dataset")
}

func (d *evaluationData) GetCheckBlock(addr addrs.Check, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	// For now, check blocks don't contain any meaningful data and can only
	// be referenced from the testing scope within an expect_failures attribute.
	//
	// We've added them into the scope explicitly since they are referencable,
	// but we'll actually just return an error message saying they can't be
	// referenced in this context.
	var diags tfdiags.Diagnostics
	diags = diags.Append(&hcl.Diagnostic{
		Severity: hcl.DiagError,
		Summary:  "Reference to \"check\" in invalid context",
		Detail:   "The \"check\" object can only be referenced from an \"expect_failures\" attribute within a Terraform testing \"run\" block.",
		Subject:  rng.ToHCL().Ptr(),
	})
	return cty.NilVal, diags
}
