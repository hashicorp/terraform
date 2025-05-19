// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package graph

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/lang/langrefs"
	hcltest "github.com/hashicorp/terraform/internal/moduletest/hcl"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
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
	relevantVariables := make(map[string]bool)

	// First, we'll check to see which variables the run block assertions
	// reference.
	for _, reference := range n.References() {
		if addr, ok := reference.Subject.(addrs.InputVariable); ok {
			relevantVariables[addr.Name] = true
		}
	}

	// And check to see which variables the run block configuration references.
	for name := range run.ModuleConfig.Module.Variables {
		relevantVariables[name] = true
	}

	// We'll put the parsed values into this map.
	values := make(terraform.InputValues)

	// First, let's step through the expressions within the run block and work
	// them out.
	for name, expr := range run.Config.Variables {
		requiredValues := make(map[string]cty.Value)

		refs, refDiags := langrefs.ReferencesInExpr(addrs.ParseRefFromTestingScope, expr)
		for _, ref := range refs {
			if addr, ok := ref.Subject.(addrs.InputVariable); ok {
				cache := ctx.GetCache(run)

				value, valueDiags := cache.GetFileVariable(addr.Name)
				diags = diags.Append(valueDiags)
				if value != nil {
					requiredValues[addr.Name] = value.Value
					continue
				}

				// Otherwise, it might be a global variable.
				value, valueDiags = cache.GetGlobalVariable(addr.Name)
				diags = diags.Append(valueDiags)
				if value != nil {
					requiredValues[addr.Name] = value.Value
					continue
				}
			}
		}
		diags = diags.Append(refDiags)

		ctx, ctxDiags := hcltest.EvalContext(hcltest.TargetRunBlock, map[string]hcl.Expression{name: expr}, requiredValues, ctx.GetOutputs())
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

	for variable := range relevantVariables {
		if _, exists := values[variable]; exists {
			// Then we've already got a value for this variable.
			continue
		}

		// Otherwise, we'll get it from the cache as a file-level or global
		// variable.
		cache := ctx.GetCache(run)

		value, valueDiags := cache.GetFileVariable(variable)
		diags = diags.Append(valueDiags)
		if value != nil {
			values[variable] = value
			continue
		}

		value, valueDiags = cache.GetGlobalVariable(variable)
		diags = diags.Append(valueDiags)
		if value != nil {
			values[variable] = value
			continue
		}
	}

	// Finally, we check the configuration again. This is where we'll discover
	// if there's any missing variables and fill in any optional variables that
	// don't have a value already.

	for name, variable := range run.ModuleConfig.Module.Variables {
		if _, exists := values[name]; exists {
			// Then we've provided a variable for this. It's all good.
			continue
		}

		// Otherwise, we're going to give these variables a value. They'll be
		// processed by the Terraform graph and provided a default value later
		// if they have one.

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

// AddVariablesToConfig extends the provided config to ensure it has definitions
// for all specified variables.
//
// This function is essentially the opposite of FilterVariablesToConfig which
// makes the variables match the config rather than the config match the
// variables.
func (n *NodeTestRun) AddVariablesToConfig(variables terraform.InputValues) {
	run := n.run
	// If we have got variable values from the test file we need to make sure
	// they have an equivalent entry in the configuration. We're going to do
	// that dynamically here.

	// First, take a backup of the existing configuration so we can easily
	// restore it later.
	currentVars := make(map[string]*configs.Variable)
	for name, variable := range run.ModuleConfig.Module.Variables {
		currentVars[name] = variable
	}

	for name, value := range variables {
		if _, exists := run.ModuleConfig.Module.Variables[name]; exists {
			continue
		}

		run.ModuleConfig.Module.Variables[name] = &configs.Variable{
			Name:           name,
			Type:           value.Value.Type(),
			ConstraintType: value.Value.Type(),
			DeclRange:      value.SourceRange.ToHCL(),
		}
	}

}
