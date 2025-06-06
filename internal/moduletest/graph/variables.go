// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package graph

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/lang/langrefs"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// GetVariables builds the terraform.InputValues required for the provided run
// block. It pulls the relevant variables (ie. the variables needed for the
// run block) from the total pool of all available variables, and converts them
// into input values.
//
// As a run block can reference variables defined within the file and are not
// actually defined within the configuration, this function actually returns
// more variables than are required by the config. FilterVariablesToConfig
// should be called before trying to use these variables within a Terraform
// plan, apply, or destroy operation.
func (n *NodeTestRun) GetVariables(ctx *EvalContext, includeWarnings bool) (terraform.InputValues, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	run := n.run
	// relevantVariables contains the variables that are of interest to this
	// run block. This is a combination of the variables declared within the
	// configuration for this run block, and the variables referenced by the
	// run block assertions.
	relevantVariables := make(map[string]*addrs.Reference)

	// First, we'll check to see which variables the run block assertions
	// reference.
	for _, reference := range n.References() {
		if addr, ok := reference.Subject.(addrs.InputVariable); ok {
			relevantVariables[addr.Name] = reference
		}
	}

	// And check to see which variables the run block configuration references.
	for name := range run.ModuleConfig.Module.Variables {
		relevantVariables[name] = nil
	}

	// We'll put the parsed values into this map.
	values := make(terraform.InputValues)

	// First, let's step through the expressions within the run block and work
	// them out.

	for name, expr := range run.Config.Variables {
		refs, refDiags := langrefs.ReferencesInExpr(addrs.ParseRefFromTestingScope, expr)
		diags = append(diags, refDiags...)
		if refDiags.HasErrors() {
			continue
		}

		ctx, ctxDiags := ctx.HclContext(refs)
		diags = diags.Append(ctxDiags)

		value := cty.DynamicVal
		if !ctxDiags.HasErrors() {
			var valueDiags hcl.Diagnostics
			value, valueDiags = expr.Value(ctx)
			diags = diags.Append(valueDiags)
		}

		// We do this late on so we still validate whatever it was that the user
		// wrote in the variable expression. But, we don't want to actually use
		// it if it's not actually relevant.
		if _, exists := relevantVariables[name]; !exists {
			// Do not display warnings during cleanup phase
			if includeWarnings {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagWarning,
					Summary:  "Value for undeclared variable",
					Detail:   fmt.Sprintf("The module under test does not declare a variable named %q, but it is declared in run block %q.", name, run.Name),
					Subject:  expr.Range().Ptr(),
				})
			}
			continue // Don't add it to our final set of variables.
		}

		values[name] = &terraform.InputValue{
			Value:       value,
			SourceType:  terraform.ValueFromConfig,
			SourceRange: tfdiags.SourceRangeFromHCL(expr.Range()),
		}
	}

	// Second, let's see if we have any variables defined with the configuration
	// we're about to test. We'll check if we have matching variable values
	// defined within the test file or globally that can match them and, if not,
	// use a default fallback value to let Terraform attempt to apply defaults
	// if they exist.

	for name, variable := range run.ModuleConfig.Module.Variables {
		if _, exists := values[name]; exists {
			// Then we've provided a variable for this explicitly. It's all
			// good.
			continue
		}

		// The user might have provided a value for this externally or at the
		// file level, so we can also just pass it through.

		if value, ok := ctx.GetVariable(variable.Name); ok {
			values[name] = value
			continue
		}
		if value, valueDiags := ctx.EvaluateUnparsedVariable(name, variable); value != nil {
			diags = diags.Append(valueDiags)
			values[name] = value
			continue
		}

		// If all else fails, these variables may have default values set within
		// the to-be-executed Terraform config. We'll put in placeholder values
		// if that is the case, otherwise add a diagnostic early to avoid
		// executing something we know will fail.

		if variable.Required() {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "No value for required variable",
				Detail: fmt.Sprintf("The module under test for run block %q has a required variable %q with no set value. Use a -var or -var-file command line argument or add this variable into a \"variables\" block within the test file or run block.",
					run.Name, variable.Name),
				Subject: variable.DeclRange.Ptr(),
			})

			values[name] = &terraform.InputValue{
				Value:       cty.DynamicVal,
				SourceType:  terraform.ValueFromConfig,
				SourceRange: tfdiags.SourceRangeFromHCL(variable.DeclRange),
			}
		} else {
			values[name] = &terraform.InputValue{
				Value:       cty.NilVal,
				SourceType:  terraform.ValueFromConfig,
				SourceRange: tfdiags.SourceRangeFromHCL(variable.DeclRange),
			}
		}
	}

	// Finally, in this whole thing we might have other variables that are
	// just referenced by parts of this run block. These must have been defined
	// elsewhere, but we need to include them.

	for variable, reference := range relevantVariables {
		if _, exists := values[variable]; exists {
			// Then we've already got a value for this variable.
			continue
		}

		// Otherwise, we'll get it from the cache as a file-level or global
		// variable.

		if value, ok := ctx.GetVariable(variable); ok {
			values[variable] = value
			continue
		}

		if reference == nil {
			// this shouldn't happen, we only put nil references into the
			// relevantVariables map for values derived from the configuration
			// and all of these should have been set in previous for loop.
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Missing reference",
				Detail:   fmt.Sprintf("The variable %q had no point of reference, which should not be possible. This is a bug in Terraform; please report it!", variable),
			})
			continue
		}

		if value, valueDiags := ctx.EvaluateUnparsedVariableDeprecated(variable, reference); value != nil {
			values[variable] = value
			diags = diags.Append(valueDiags)
		}
	}

	return values, diags
}

// FilterVariablesToModule splits the provided values into two disjoint maps:
// moduleVars contains the ones that correspond with declarations in the root
// module of the given configuration, while testOnlyVars contains any others
// that are presumably intended only for use in the test configuration file.
//
// This function is essentially the opposite of AddVariablesToConfig which
// makes the config match the variables rather than the variables match the
// config.
//
// This function can only return warnings, and the callers can rely on this so
// please check the callers of this function if you add any error diagnostics.
func (n *NodeTestRun) FilterVariablesToModule(values terraform.InputValues) (moduleVars, testOnlyVars terraform.InputValues, diags tfdiags.Diagnostics) {
	moduleVars = make(terraform.InputValues)
	testOnlyVars = make(terraform.InputValues)
	for name, value := range values {
		_, exists := n.run.ModuleConfig.Module.Variables[name]
		if !exists {
			// If it's not in the configuration then it's a test-only variable.
			testOnlyVars[name] = value
			continue
		}

		moduleVars[name] = value
	}
	return moduleVars, testOnlyVars, diags
}
