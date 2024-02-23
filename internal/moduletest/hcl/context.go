// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package hcl

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type EvalContextTarget string

const (
	TargetRunBlock EvalContextTarget = "run"
	TargetProvider EvalContextTarget = "provider"
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
//
// We perform some pre-validation of the expected expressions that this context
// will be used to evaluate. This is just so we can provide some better error
// messages and diagnostics. The expressions argument could be empty without
// affecting the returned context.
func EvalContext(target EvalContextTarget, expressions []hcl.Expression, availableVariables map[string]cty.Value, availableRunOutputs map[addrs.Run]cty.Value) (*hcl.EvalContext, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	runs := make(map[string]cty.Value, len(availableRunOutputs))
	for addr, objVal := range availableRunOutputs {
		runs[addr.Name] = objVal
	}

	for _, expression := range expressions {
		refs, refDiags := lang.ReferencesInExpr(addrs.ParseRefFromTestingScope, expression)
		diags = diags.Append(refDiags)

		for _, ref := range refs {
			if addr, ok := ref.Subject.(addrs.Run); ok {
				objVal, exists := availableRunOutputs[addr]

				var diagPrefix string
				switch target {
				case TargetRunBlock:
					diagPrefix = "You can only reference run blocks that are in the same test file and will execute before the current run block."
				case TargetProvider:
					diagPrefix = "You can only reference run blocks that are in the same test file and will execute before the provider is required."
				}

				if !exists {
					// Then this is a made up run block.
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Reference to unknown run block",
						Detail:   fmt.Sprintf("The run block %q does not exist within this test file. %s", addr.Name, diagPrefix),
						Subject:  ref.SourceRange.ToHCL().Ptr(),
					})

					continue
				}

				if objVal == cty.NilVal {
					// This run block exists, but it is after the current run block.
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Reference to unavailable run block",
						Detail:   fmt.Sprintf("The run block %q has not executed yet. %s", addr.Name, diagPrefix),
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
					//
					// When reporting this we assume that it's happened because
					// the prior run was a plan-only run and that some of its
					// output values were not known. If this arises for a
					// run that performed a full apply then this is a bug in
					// Terraform's modules runtime, because unknown output
					// values should not be possible in that case.
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Reference to unknown value",
						Detail:   fmt.Sprintf("The value for %s is unknown. Run block %q is executing a \"plan\" operation, and the specified output value is only known after apply.", ref.DisplayString(), addr.Name),
						Subject:  ref.SourceRange.ToHCL().Ptr(),
					})
					continue
				}

				continue
			}

			if addr, ok := ref.Subject.(addrs.InputVariable); ok {
				if _, exists := availableVariables[addr.Name]; !exists {
					// This variable reference doesn't exist.

					detail := fmt.Sprintf("The input variable %q is not available to the current context. Within the variables block of a run block you can only reference variables defined at the file or global levels; within the variables block of a suite you can only reference variables defined at the global levels.", addr.Name)
					if availableRunOutputs == nil {
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
			if availableRunOutputs == nil {
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
			if availableRunOutputs != nil {
				variables["run"] = cty.ObjectVal(runs)
			}
			return variables
		}(),
		Functions: lang.TestingFunctions(),
	}, diags
}
