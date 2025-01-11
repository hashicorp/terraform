// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package hcl

import (
	"sync"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/backend/backendrun"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// TestContext contains a mapping between test run blocks and evaluated
// variables. This is used to cache the results of evaluating variables so that
// they are only evaluated once per run.
//
// Each run block has its own configuration and therefore its own set of
// evaluated variables.
type TestContext struct {
	GlobalVariables map[string]backendrun.UnparsedVariableValue
	globalLock      sync.Mutex
	FileVariables   map[string]hcl.Expression
	fileLock        sync.Mutex

	Config *configs.Config

	ParsedGlobalVariables  terraform.InputValues
	ParsedGlobalVariables2 map[string]terraform.InputValues
	ParsedFileVariables    terraform.InputValues
	ConfigVariables        terraform.InputValues
	ConfigVariables2       map[string]terraform.InputValues
	RunVariables           map[string]terraform.InputValues

	// RunOutputs is a mapping from run addresses to cty object values
	// representing the collected output values from the module under test.
	//
	// This is used to allow run blocks to refer back to the output values of
	// previous run blocks. It is passed into the Evaluate functions that
	// validate the test assertions, and used when calculating values for
	// variables within run blocks.
	RunOutputs map[addrs.Run]cty.Value
}

func NewTestContext(config *configs.Config, globalVariables map[string]backendrun.UnparsedVariableValue, fileVariables map[string]hcl.Expression, runOutputs map[addrs.Run]cty.Value) *TestContext {
	return &TestContext{
		Config:                 config,
		GlobalVariables:        globalVariables,
		FileVariables:          fileVariables,
		ParsedGlobalVariables:  make(terraform.InputValues),
		ParsedGlobalVariables2: make(map[string]terraform.InputValues),
		ParsedFileVariables:    make(terraform.InputValues),
		ConfigVariables:        make(terraform.InputValues),
		ConfigVariables2:       make(map[string]terraform.InputValues),
		RunVariables:           make(map[string]terraform.InputValues),
		RunOutputs:             runOutputs,
	}
}

func (cache *TestContext) SetGlobalVariable(name string, value *terraform.InputValue) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	_, exists := cache.GlobalVariables[name]
	if !exists {
		return nil // TODO: return an error
	}
	cache.ParsedGlobalVariables[name] = value
	return diags
}

func (cache *TestContext) SetFileVariable(name string, value *terraform.InputValue) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	_, exists := cache.FileVariables[name]
	if !exists {
		return nil // TODO: return an error
	}
	cache.ParsedFileVariables[name] = value
	return diags
}

func (cache *TestContext) SetRunVariable(runName, varName string, value *terraform.InputValue) tfdiags.Diagnostics {
	store, exists := cache.RunVariables[runName]
	if !exists {
		store = make(terraform.InputValues)
		cache.RunVariables[runName] = store
	}
	cache.RunVariables[runName][varName] = value
	return nil
}

func (cache *TestContext) GetGlobalVariable(name string) (*terraform.InputValue, tfdiags.Diagnostics) {
	_, exists := cache.GlobalVariables[name]
	if !exists {
		return nil, nil
	}

	return cache.ParsedGlobalVariables[name], nil
}

func (cache *TestContext) GetFileVariable(name string) (*terraform.InputValue, tfdiags.Diagnostics) {
	_, exists := cache.FileVariables[name]
	if !exists {
		return nil, nil
	}

	return cache.ParsedFileVariables[name], nil
}

func (cache *TestContext) GetRunVariable(runName, varName string) (*terraform.InputValue, tfdiags.Diagnostics) {
	store, exists := cache.RunVariables[runName]
	if !exists {
		return nil, nil
	}
	return store[varName], nil
}

func (cache *TestContext) GetParsedVariables(key, runName string) terraform.InputValues {
	variables := make(terraform.InputValues)
	// The order of these assignments is important. The variables from the
	// config are the lowest priority and should be overridden by the variables
	// from the parsed files and global variables.
	if configVariables, exists := cache.ConfigVariables2[key]; exists {
		for name, value := range configVariables {
			variables[name] = value
		}
	}

	for name, value := range cache.ParsedGlobalVariables {
		variables[name] = value
	}

	for name, value := range cache.ParsedFileVariables {
		variables[name] = value
	}

	if runVariables, exists := cache.RunVariables[runName]; exists {
		for name, value := range runVariables {
			variables[name] = value
		}
	}

	return variables
}

func (cache *TestContext) GetConfigVariable(name string) (*terraform.InputValue, tfdiags.Diagnostics) {
	value, exists := cache.ConfigVariables[name]
	if !exists {
		return nil, nil
	}

	return value, nil
}

func (cache *TestContext) SetConfigVariable(name string, value *terraform.InputValue) tfdiags.Diagnostics {
	cache.ConfigVariables[name] = value
	return nil
}

func (cache *TestContext) GetConfigVariable2(key, name string) (*terraform.InputValue, tfdiags.Diagnostics) {
	mp, exists := cache.ConfigVariables2[key]
	if !exists {
		return nil, nil
	}
	value, exists := mp[name]
	if !exists {
		return nil, nil
	}

	return value, nil
}

func (cache *TestContext) SetConfigVariable2(key, name string, value *terraform.InputValue) tfdiags.Diagnostics {
	mp, exists := cache.ConfigVariables2[key]
	if !exists {
		mp = make(terraform.InputValues)
		cache.ConfigVariables2[key] = mp
	}
	mp[name] = value
	return nil
}

func (cache *TestContext) GetGlobalVariable2(key, name string) (*terraform.InputValue, tfdiags.Diagnostics) {
	mp, exists := cache.ParsedGlobalVariables2[key]
	if !exists {
		return nil, nil
	}
	value, exists := mp[name]
	if !exists {
		return nil, nil
	}

	return value, nil
}

func (cache *TestContext) SetGlobalVariable2(key, name string, value *terraform.InputValue) tfdiags.Diagnostics {
	mp, exists := cache.ParsedGlobalVariables2[key]
	if !exists {
		mp = make(terraform.InputValues)
		cache.ParsedGlobalVariables2[key] = mp
	}
	mp[name] = value
	return nil
}

// GetGlobalVariable returns a value for the named global variable evaluated
// against the current run.
//
// This function caches the result of evaluating the variable so that it is
// only evaluated once per run.
//
// This function will return a valid input value if parsing fails for any reason
// so the caller can continue processing the configuration. The diagnostics
// returned will contain the error message that occurred during parsing and as
// such should be shown to the user.
// func (cache *VariableCaches2) GetGlobalVariable(name string) (*terraform.InputValue, tfdiags.Diagnostics) {
// 	variable, exists := cache.GlobalVariables[name]
// 	if !exists {
// 		return nil, nil
// 	}

// 	// TODO: We should also introduce a way to specify the mode in the test
// 	//   file itself. Suggestion, optional variable blocks.
// 	parsingMode := configs.VariableParseHCL

// 	if cfg, exists := cache.config.Module.Variables[name]; exists {
// 		parsingMode = cfg.ParsingMode
// 	}

// 	value, diags := variable.ParseVariableValue(parsingMode)
// 	if diags.HasErrors() {
// 		// In this case, the variable exists but we couldn't parse it. We'll
// 		// return a usable value so that we don't compound errors later by
// 		// claiming a variable doesn't exist when it does. We also return the
// 		// diagnostics explaining the error which will be shown to the user.
// 		value = &terraform.InputValue{
// 			Value: cty.DynamicVal,
// 		}
// 	}

// 	cache.ParsedFileVariables[name] = value
// 	return value, diags
// }

// // GetFileVariable returns a value for the named file-level variable evaluated
// // against the current run.
// //
// // This function caches the result of evaluating the variable so that it is
// // only evaluated once per run.
// //
// // This function will return a valid input value if parsing fails for any reason
// // so the caller can continue processing the configuration. The diagnostics
// // returned will contain the error message that occurred during parsing and as
// // such should be shown to the user.
// func (cache *VariableCaches2) GetFileVariable(name string) (*terraform.InputValue, tfdiags.Diagnostics) {
// 	var diags tfdiags.Diagnostics

// 	availableVariables := make(map[string]cty.Value)
// 	// If we had referenced a global variable in the file variable, we need to
// 	// get it from the global variables store. e.g. `var.foo` in the file var
// 	refs, refDiags := langrefs.ReferencesInExpr(addrs.ParseRefFromTestingScope, expr)
// 	for _, ref := range refs {
// 		if input, ok := ref.Subject.(addrs.InputVariable); ok {
// 			variable, variableDiags := cache.GetGlobalVariable(input.Name)
// 			diags = diags.Append(variableDiags)
// 			if variable != nil {
// 				availableVariables[input.Name] = variable.Value
// 			}
// 		}
// 	}
// 	diags = diags.Append(refDiags)

// 	if diags.HasErrors() {
// 		// There's no point trying to evaluate the variable as we know it will
// 		// fail. We'll just return a usable value so that we don't compound
// 		// errors later by claiming a variable doesn't exist when it does. We
// 		// also return the diagnostics explaining the error which will be shown
// 		// to the user.
// 		cache.files[name] = &terraform.InputValue{
// 			Value: cty.DynamicVal,
// 		}
// 		return cache.files[name], diags
// 	}

// 	ctx, ctxDiags := EvalContext(TargetFileVariable, map[string]hcl.Expression{name: expr}, availableVariables, nil)
// 	diags = diags.Append(ctxDiags)

// 	if ctxDiags.HasErrors() {
// 		// If we couldn't build the context, we won't actually process these
// 		// variables. Instead, we'll fill them with an empty value but still
// 		// make a note that the user did provide them.
// 		cache.files[name] = &terraform.InputValue{
// 			Value: cty.DynamicVal,
// 		}
// 		return cache.files[name], diags
// 	}

// 	value, valueDiags := expr.Value(ctx)
// 	diags = diags.Append(valueDiags)
// 	if diags.HasErrors() {
// 		// In this case, the variable exists but we couldn't parse it. We'll
// 		// return a usable value so that we don't compound errors later by
// 		// claiming a variable doesn't exist when it does. We also return the
// 		// diagnostics explaining the error which will be shown to the user.
// 		value = cty.DynamicVal
// 	}

// 	cache.files[name] = &terraform.InputValue{
// 		Value:       value,
// 		SourceType:  terraform.ValueFromConfig,
// 		SourceRange: tfdiags.SourceRangeFromHCL(expr.Range()),
// 	}
// 	return cache.files[name], diags
// }
