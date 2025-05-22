// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package hcl

import (
	"fmt"
	"sync"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/backend/backendrun"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/lang/langrefs"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type VariableCache struct {
	mutex sync.Mutex

	// ExternalVariableValues contains the raw values provided by the user
	// via either the CLI, environment variables, or a variable file.
	ExternalVariableValues map[string]backendrun.UnparsedVariableValue

	// TestFileVariableDefinitions contains the definitions for variables
	// defined within the test file in `variable` blocks.
	TestFileVariableDefinitions map[string]*configs.Variable

	// TestFileVariableExpressions contains the concrete variable expressions
	// defined within the test file `variables` block.
	TestFileVariableExpressions map[string]hcl.Expression

	// fileVariableValues contains the set of available file level
	fileVariableValues map[string]*terraform.InputValue
}

func (cache *VariableCache) EvaluateExternalVariable(name string, config *configs.Variable) (*terraform.InputValue, tfdiags.Diagnostics) {
	variable, exists := cache.ExternalVariableValues[name]
	if !exists {
		return nil, nil
	}

	if config != nil {

		// If we have a configuration, then we'll using the parsing mode from
		// that.

		value, diags := variable.ParseVariableValue(config.ParsingMode)
		if diags.HasErrors() {
			value = &terraform.InputValue{
				Value: cty.DynamicVal,
			}
		}
		return value, diags
	}

	// For backwards-compatibility reasons we do also have to support trying
	// to parse the global variables without a configuration. We introduced the
	// file-level variable definitions later, and users were already using
	// global variables so we do need to keep supporting this use case.

	// Otherwise, we have no configuration so we're going to try both parsing
	// modes.

	value, diags := variable.ParseVariableValue(configs.VariableParseHCL)
	if !diags.HasErrors() {
		// then good! we can just return these values directly.
		return value, diags
	}

	// otherwise, we'll try the other one.

	value, diags = variable.ParseVariableValue(configs.VariableParseLiteral)
	if diags.HasErrors() {

		// we'll add a warning diagnostic here, just telling the users they
		// can avoid this by adding a variable definition.

		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Warning,
			"Missing variable definition",
			fmt.Sprintf("The variable %q could not be parsed. Terraform had no definition block for this variable, this error could be avoided in future by including a definition block for this variable within the Terraform test file.", name)))

		// as usual make sure we still provide something for this value.

		value = &terraform.InputValue{
			Value: cty.DynamicVal,
		}
	}
	return value, diags
}

func (cache *VariableCache) evaluateVariableDefinition(name string) (*terraform.InputValue, tfdiags.Diagnostics) {
	definition, exists := cache.TestFileVariableDefinitions[name]
	if !exists {
		// no definition for this variable
		return nil, nil
	}

	var diags tfdiags.Diagnostics

	var input *terraform.InputValue
	if _, exists := cache.ExternalVariableValues[name]; exists {
		parsed, moreDiags := cache.EvaluateExternalVariable(name, definition)
		diags = diags.Append(moreDiags)
		input = parsed
	} else {
		input = &terraform.InputValue{
			Value: cty.NilVal,
		}
	}

	value, moreDiags := terraform.PrepareFinalInputVariableValue(addrs.AbsInputVariableInstance{
		Module: addrs.RootModuleInstance,
		Variable: addrs.InputVariable{
			Name: name,
		},
	}, input, definition)
	diags = diags.Append(moreDiags)

	return &terraform.InputValue{
		Value:       value,
		SourceType:  terraform.ValueFromConfig,
		SourceRange: tfdiags.SourceRangeFromHCL(definition.DeclRange),
	}, diags
}

func (cache *VariableCache) evaluateFileVariable(name string) (*terraform.InputValue, tfdiags.Diagnostics) {
	expr, exists := cache.TestFileVariableExpressions[name]
	if !exists {
		return nil, nil
	}

	var diags tfdiags.Diagnostics

	availableVariables := make(map[string]cty.Value)
	refs, refDiags := langrefs.ReferencesInExpr(addrs.ParseRefFromTestingScope, expr)
	for _, ref := range refs {
		if input, ok := ref.Subject.(addrs.InputVariable); ok {
			variable, variableDiags := cache.evaluateVariableDefinition(input.Name)
			if variable != nil {
				diags = diags.Append(variableDiags)
				availableVariables[input.Name] = variable.Value
			} else if variable, variableDiags := cache.EvaluateExternalVariable(input.Name, nil); variable != nil {
				diags = diags.Append(variableDiags)
				availableVariables[input.Name] = variable.Value
			}
		}
	}
	diags = diags.Append(refDiags)

	if diags.HasErrors() {
		return &terraform.InputValue{
			Value: cty.DynamicVal,
		}, diags
	}

	ctx, ctxDiags := EvalContext(TargetFileVariable, map[string]hcl.Expression{name: expr}, availableVariables, nil)
	diags = diags.Append(ctxDiags)
	if ctxDiags.HasErrors() {
		return &terraform.InputValue{
			Value: cty.DynamicVal,
		}, diags
	}

	value, valueDiags := expr.Value(ctx)
	diags = diags.Append(valueDiags)
	if diags.HasErrors() {
		// In this case, the variable exists but we couldn't parse it. We'll
		// return a usable value so that we don't compound errors later by
		// claiming a variable doesn't exist when it does. We also return the
		// diagnostics explaining the error which will be shown to the user.
		value = cty.DynamicVal
	}

	return &terraform.InputValue{
		Value:       value,
		SourceType:  terraform.ValueFromConfig,
		SourceRange: tfdiags.SourceRangeFromHCL(expr.Range()),
	}, diags
}

func (cache *VariableCache) GetVariableValue(name string) (*terraform.InputValue, tfdiags.Diagnostics) {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()

	if cache.fileVariableValues == nil {
		cache.fileVariableValues = make(map[string]*terraform.InputValue)
	}

	if value, exists := cache.fileVariableValues[name]; exists {
		return value, nil
	}

	if value, valueDiags := cache.evaluateFileVariable(name); value != nil {
		cache.fileVariableValues[name] = value
		return value, valueDiags
	}

	if value, valueDiags := cache.evaluateVariableDefinition(name); value != nil {
		cache.fileVariableValues[name] = value
		return value, valueDiags
	}

	if value, valueDiags := cache.EvaluateExternalVariable(name, nil); value != nil {
		cache.fileVariableValues[name] = value
		return value, valueDiags
	}

	return nil, nil
}

func (cache *VariableCache) HasVariableDefinition(name string) bool {
	if _, exists := cache.TestFileVariableExpressions[name]; exists {
		return true
	}

	if _, exists := cache.TestFileVariableDefinitions[name]; exists {
		return true
	}
	return false
}
