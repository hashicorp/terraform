// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package hcl

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// EvalContext builds hcl.EvalContext objects for use directly within the
// testing framework.
//
// We support referencing variables from the file variables block, and any
// global variables provided via the CLI / environment variables / .tfvars
// files. These should be provided in the availableVariables argument, already
// parsed and ready for use.
//
// We also support referencing outputs from any previous run blocks. These
// should be provided in the availableRunBlocks argument. As we also perform
// validation (see below) the format of this argument matters. If it is
// completely null, then we do not support the `run` argument at all in this
// context. If a run block is not present at all, then we should return a "run
// block does not exist" error. If the run block is present, but contains a
// nil context, then we should return a "run block has not yet executed" error.
// Finally, if the run block is present and contains a valid value we should
// use that value in the returned HCL contexts.
//
// As referenced above, this function performs pre-validation to make sure the
// expressions to be evaluated will pass evaluation. Anything present in the
// expressions argument will be validated to make sure the only reference the
// availableVariables and availableRunBlocks.
func EvalContext(expressions []hcl.Expression, availableVariables map[string]cty.Value, availableRunBlocks map[string]*terraform.TestContext) (*hcl.EvalContext, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	runs := make(map[string]cty.Value)
	for name, ctx := range availableRunBlocks {
		if ctx == nil {
			// Then this is a valid run block, but it hasn't executed yet so we
			// won't take any values from it.
			continue
		}
		outputs := make(map[string]cty.Value)
		for name, config := range ctx.Config.Module.Outputs {
			output := ctx.State.OutputValue(addrs.AbsOutputValue{
				OutputValue: addrs.OutputValue{
					Name: name,
				},
				Module: addrs.RootModuleInstance,
			})

			var value cty.Value
			switch {
			case output == nil:
				// This means the run block returned null for this output.
				// It is likely this will produce an error later if it is
				// referenced, but users can actually specify that null
				// is an acceptable value for an input variable so we won't
				// actually raise a fuss about this at all.
				value = cty.NullVal(cty.DynamicPseudoType)
			case output.Value.IsNull() || output.Value == cty.NilVal:
				// This means the output value was returned as (known after
				// apply). If this is referenced it always an error, we
				// can't handle this in an appropriate way at all. For now,
				// we just mark it as unknown and then later we check and
				// resolve all the references. We'll raise an error at that
				// point if the user actually attempts to reference a value
				// that is unknown.
				value = cty.DynamicVal
			default:
				value = output.Value
			}

			if config.Sensitive || (output != nil && output.Sensitive) {
				value = value.Mark(marks.Sensitive)
			}

			outputs[name] = value
		}

		runs[name] = cty.ObjectVal(outputs)
	}

	for _, expression := range expressions {
		refs, refDiags := lang.ReferencesInExpr(addrs.ParseRefFromTestingScope, expression)
		diags = diags.Append(refDiags)

		for _, ref := range refs {
			if addr, ok := ref.Subject.(addrs.Run); ok {

				if availableRunBlocks == nil {
					// Then run blocks are never available from this context.
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Invalid reference",
						Detail:   "You cannot reference run blocks from within provider configurations. You can only reference run blocks from other run blocks that execute after them.",
						Subject:  ref.SourceRange.ToHCL().Ptr(),
					})

					continue
				}

				// For the error messages here, we know the reference is coming
				// from a run block as that is the only place that reference
				// other run blocks.

				ctx, exists := availableRunBlocks[addr.Name]

				if !exists {
					// Then this is a made up run block.
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Reference to unknown run block",
						Detail:   fmt.Sprintf("The run block %q does not exist within this test file. You can only reference run blocks that are in the same test file and will execute before the current run block.", addr.Name),
						Subject:  ref.SourceRange.ToHCL().Ptr(),
					})

					continue
				}

				if ctx == nil {
					// This run block exists, but it is after the current run block.
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Reference to unavailable run block",
						Detail:   fmt.Sprintf("The run block %q is not available to the current run block. You can only reference run blocks that are in the same test file and will execute before the current run block.", addr.Name),
						Subject:  ref.SourceRange.ToHCL().Ptr(),
					})

					continue
				}

				value, valueDiags := ref.Remaining.TraverseRel(runs[addr.Name])
				diags = diags.Append(valueDiags)
				if valueDiags.HasErrors() {
					// This means the reference was invalid somehow, we've
					// already added the errors to our diagnostics though so
					// we'll just carry on.
					continue
				}

				if !value.IsWhollyKnown() {
					// This is not valid, we cannot allow users to pass unknown
					// values into run blocks. There's just going to be
					// difficult and confusing errors later if this happens.

					if ctx.Run.Config.Command == configs.PlanTestCommand {
						// Then the user has likely attempted to use an output
						// that is (known after apply) due to the referenced
						// run block only being a plan command.
						diags = diags.Append(&hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  "Reference to unknown value",
							Detail:   fmt.Sprintf("The value for %s is unknown. Run block %q is executing a \"plan\" operation, and the specified output value is only known after apply.", ref.DisplayString(), addr.Name),
							Subject:  ref.SourceRange.ToHCL().Ptr(),
						})

						continue
					}

					// Otherwise, this is a bug in Terraform. We shouldn't be
					// producing (known after apply) values during apply
					// operations.
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Reference to unknown value",
						Detail:   fmt.Sprintf("The value for %s is unknown; This is a bug in Terraform, please report it.", ref.DisplayString()),
						Subject:  ref.SourceRange.ToHCL().Ptr(),
					})
				}

				continue
			}

			if addr, ok := ref.Subject.(addrs.InputVariable); ok {
				if _, exists := availableVariables[addr.Name]; !exists {
					// This variable reference doesn't exist.

					detail := fmt.Sprintf("The input variable %q is not available to the current run block. You can only reference variables defined at the file or global levels when populating the variables block within a run block.", addr.Name)
					if availableRunBlocks == nil {
						detail = fmt.Sprintf("The input variable %q is not available to the current provider configuration. You can only reference variables defined at the file or global levels within provider configurations.", addr.Name)
					}

					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Reference to unavailable variable",
						Detail:   detail,
						Subject:  ref.SourceRange.ToHCL().Ptr(),
					})

					continue
				}

				// Otherwise, we're good. This is an acceptable reference.
				continue
			}

			detail := "You can only reference earlier run blocks, file level, and global variables while defining variables from inside a run block."
			if availableRunBlocks == nil {
				detail = "You can only reference file level and global variables from inside provider configurations within test files."
			}

			// You can only reference run blocks and variables from the run
			// block variables.
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid reference",
				Detail:   detail,
				Subject:  ref.SourceRange.ToHCL().Ptr(),
			})
		}
	}

	return &hcl.EvalContext{
		Variables: func() map[string]cty.Value {
			variables := make(map[string]cty.Value)
			variables["var"] = cty.ObjectVal(availableVariables)
			if availableRunBlocks != nil {
				variables["run"] = cty.ObjectVal(runs)
			}
			return variables
		}(),
	}, diags
}
