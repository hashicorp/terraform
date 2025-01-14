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
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type VariableWithDiag struct {
	Value *terraform.InputValue
	Diags tfdiags.Diagnostics
}

// func newVariableWithDiag() VariableWithDiag {
// 	return VariableWithDiag{
// 		Value: make(terraform.InputValues),
// 	}
// }

// VariableContext holds the context for variables used in tests.
type VariableContext struct {
	// Config is the configuration for the test.
	Config *configs.Config

	// GlobalVariables contains the unparsed values for global variables.
	GlobalVariables map[string]backendrun.UnparsedVariableValue

	// FileVariables contains the HCL expressions for file-level variables.
	FileVariables map[string]hcl.Expression

	// ParsedGlobalVariables contains the evaluated values for global variables.
	ParsedGlobalVariables terraform.InputValues
	globalLock            sync.RWMutex

	// ParsedFileVariables contains the evaluated values for file-level variables.
	ParsedFileVariables map[string]VariableWithDiag
	fileLock            sync.RWMutex

	// ConfigVariables contains the evaluated values for variables declared in the
	// configuration.
	// Each key is the source directory of the module that the variables are
	// declared in.
	ConfigVariables map[string]map[string]VariableWithDiag
	configLock      sync.RWMutex

	// RunVariables contains the evaluated values for variables declared in the
	// run blocks.
	RunVariables map[string]map[string]VariableWithDiag
	runLock      sync.RWMutex

	// RunOutputs is a mapping from run addresses to cty object values
	// representing the collected output values from the module under test.
	//
	// This is used to allow run blocks to refer back to the output values of
	// previous run blocks. It is passed into the Evaluate functions that
	// validate the test assertions and is used when calculating values for
	// variables within run blocks.
	RunOutputs map[addrs.Run]cty.Value
	outputLock sync.RWMutex

	// ConfigProviders is a cache of config keys mapped to all the providers
	// referenced by the given config.
	//
	// The config keys are globally unique across an entire test suite.
	ConfigProviders map[string]map[string]bool
	providerLock    sync.RWMutex
}

// NewTestContext creates a new VariableContext with the provided configuration and variables.
func NewTestContext(config *configs.Config, globalVariables map[string]backendrun.UnparsedVariableValue, fileVariables map[string]hcl.Expression, runOutputs map[addrs.Run]cty.Value) *VariableContext {
	return &VariableContext{
		Config:                config,
		GlobalVariables:       globalVariables,
		FileVariables:         fileVariables,
		ParsedGlobalVariables: make(terraform.InputValues),
		ParsedFileVariables:   make(map[string]VariableWithDiag),
		ConfigVariables:       make(map[string]map[string]VariableWithDiag),
		RunVariables:          make(map[string]map[string]VariableWithDiag),
		RunOutputs:            runOutputs,
		ConfigProviders:       make(map[string]map[string]bool),
	}
}

// SetGlobalVariable sets a global variable in the context.
func (cache *VariableContext) SetGlobalVariable(name string, value *terraform.InputValue) {
	cache.globalLock.Lock()
	defer cache.globalLock.Unlock()
	cache.ParsedGlobalVariables[name] = value
}

// SetFileVariable sets a file-level variable in the context.
func (cache *VariableContext) SetFileVariable(name string, value VariableWithDiag) {
	cache.fileLock.Lock()
	defer cache.fileLock.Unlock()
	cache.ParsedFileVariables[name] = value
}

// SetRunVariable sets a run-level variable in the context.
func (cache *VariableContext) SetRunVariable(runName, varName string, value VariableWithDiag) tfdiags.Diagnostics {
	// // relevantVariables contains the variables that are of interest to this
	// // run block. This is a combination of the variables declared within the
	// // configuration for this run block, and the variables referenced by the
	// // run block assertions.
	// relevantVariables := make(map[string]bool)

	// // First, we'll check to see which variables the run block assertions
	// // reference.
	// runRefs, diags := run.GetReferences()
	// if diags.HasErrors() {
	// 	return diags
	// }
	// for _, reference := range runRefs {
	// 	if addr, ok := reference.Subject.(addrs.InputVariable); ok {
	// 		relevantVariables[addr.Name] = true
	// 	}
	// }

	// // If we're testing a specific configuration, we need to use that
	// if run.Config.ConfigUnderTest != nil {
	// 	config = run.Config.ConfigUnderTest
	// }

	// // And check to see which variables the run block configuration references.
	// for name := range config.Module.Variables {
	// 	relevantVariables[name] = true
	// }

	// requiredValues := make(map[string]cty.Value)
	// refs, refDiags := langrefs.ReferencesInExpr(addrs.ParseRefFromTestingScope, n.Expr)
	// for _, ref := range refs {
	// 	if addr, ok := ref.Subject.(addrs.InputVariable); ok {
	// 		value, valueDiags := cache.GetFileVariable(addr.Name)
	// 		diags = diags.Append(valueDiags)
	// 		if value != nil {
	// 			requiredValues[addr.Name] = value.Value
	// 			continue
	// 		}

	// 		// Otherwise, it might be a global variable.
	// 		value, valueDiags = cache.GetGlobalVariable(addr.Name)
	// 		diags = diags.Append(valueDiags)
	// 		if value != nil {
	// 			requiredValues[addr.Name] = value.Value
	// 			continue
	// 		}
	// 	}
	// }
	// diags = diags.Append(refDiags)

	// ctx, ctxDiags := EvalContext(TargetRunBlock, map[string]hcl.Expression{varName: n.Expr}, requiredValues, cache.RunOutputs)
	// diags = diags.Append(ctxDiags)

	// value := cty.DynamicVal
	// if !ctxDiags.HasErrors() {
	// 	var valueDiags hcl.Diagnostics
	// 	value, valueDiags = n.Expr.Value(ctx)
	// 	diags = diags.Append(valueDiags)
	// }

	// // We do this late on so we still validate whatever it was that the user
	// // wrote in the variable expression. But, we don't want to actually use
	// // it if it's not actually relevant.
	// if _, exists := relevantVariables[n.Addr.Name]; !exists {
	// 	// Do not display warnings during cleanup2 phase
	// 	// if includeWarnings { // TODO
	// 	diags = diags.Append(&hcl.Diagnostic{
	// 		Severity: hcl.DiagWarning,
	// 		Summary:  "Value for undeclared variable",
	// 		Detail:   fmt.Sprintf("The module under test does not declare a variable named %q, but it is declared in run block %q.", n.Addr.Name, n.run.Name),
	// 		Subject:  n.Expr.Range().Ptr(),
	// 	})
	// 	// }
	// 	return diags
	// }

	// inputValue := &terraform.InputValue{
	// 	Value:       value,
	// 	SourceType:  terraform.ValueFromConfig,
	// 	SourceRange: tfdiags.SourceRangeFromHCL(n.Expr.Range()),
	// }
	cache.runLock.Lock()
	defer cache.runLock.Unlock()
	store, exists := cache.RunVariables[runName]
	if !exists {
		store = make(map[string]VariableWithDiag)
		cache.RunVariables[runName] = store
	}
	store[varName] = value

	return nil
}

// GetGlobalVariable retrieves a global variable from the context.
func (cache *VariableContext) GetGlobalVariable(name string) (*terraform.InputValue, tfdiags.Diagnostics) {
	cache.globalLock.RLock()
	defer cache.globalLock.RUnlock()
	value, exists := cache.ParsedGlobalVariables[name]
	if !exists {
		return nil, nil
	}
	return value, nil
}

// GetFileVariable retrieves a file-level variable from the context.
func (cache *VariableContext) GetFileVariable(name string) (*terraform.InputValue, tfdiags.Diagnostics) {
	cache.fileLock.RLock()
	defer cache.fileLock.RUnlock()
	value, exists := cache.ParsedFileVariables[name]
	if !exists {
		return nil, nil
	}
	return value.Value, value.Diags
}

// GetRunVariable retrieves a run-level variable from the context.
func (cache *VariableContext) GetRunVariable(runName, varName string) (*terraform.InputValue, tfdiags.Diagnostics) {
	cache.runLock.RLock()
	defer cache.runLock.RUnlock()
	store, exists := cache.RunVariables[runName]
	if !exists {
		return nil, nil
	}
	value, exists := store[varName]
	if !exists {
		return nil, nil
	}
	return value.Value, value.Diags
}

// GetParsedVariables retrieves all parsed variables for a given key and run name.
func (cache *VariableContext) GetParsedVariables(mod *configs.Module, run *moduletest.Run) (terraform.InputValues, tfdiags.Diagnostics) {
	variables := make(terraform.InputValues)
	var diags tfdiags.Diagnostics

	relevantVariables := make(map[string]bool)

	references, refDiags := run.GetReferences()
	if refDiags.HasErrors() {
		return nil, refDiags
	}

	// First, we'll check to see which variables the run block assertions
	// reference.
	for _, reference := range references {
		if addr, ok := reference.Subject.(addrs.InputVariable); ok {
			relevantVariables[addr.Name] = true
		}
	}
	// And check to see which variables the run block configuration references.
	for name := range mod.Variables {
		relevantVariables[name] = true
	}

	cache.configLock.RLock()
	configVariables, configExists := cache.ConfigVariables[mod.SourceDir]
	cache.configLock.RUnlock()

	cache.globalLock.RLock()
	globalVariables := cache.ParsedGlobalVariables
	cache.globalLock.RUnlock()

	cache.fileLock.RLock()
	fileVariables := cache.ParsedFileVariables
	cache.fileLock.RUnlock()

	cache.runLock.RLock()
	runVariables, runExists := cache.RunVariables[run.Name]
	cache.runLock.RUnlock()

	valIsNil := func(v *terraform.InputValue) bool {
		return v == nil || v.Value.Type() == cty.NilType || v.Value.Type() == cty.DynamicPseudoType
	}

	for name := range relevantVariables {
		var value *terraform.InputValue
		var valueDiags tfdiags.Diagnostics

		if runExists {
			if v, exists := runVariables[name]; exists {
				value = v.Value
				valueDiags = v.Diags
			}
		}

		if valIsNil(value) {
			if v, exists := fileVariables[name]; exists {
				value = v.Value
				valueDiags = v.Diags
			}
		}

		if valIsNil(value) {
			if v, exists := globalVariables[name]; exists {
				value = v
			}
		}

		if valIsNil(value) && configExists {
			if v, exists := configVariables[name]; exists {
				value = v.Value
				valueDiags = v.Diags
			}
		}

		if value != nil {
			variables[name] = value
			if valueDiags.HasErrors() {
				diags = diags.Append(valueDiags)
			}
		} else {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Variable not found",
				Detail:   fmt.Sprintf("The variable %q was not found in any context.", name),
			})
		}
	}

	return variables, diags
}

// GetConfigVariable retrieves a configuration variable from the context.
func (cache *VariableContext) GetConfigVariable(mod *configs.Module, name string) (*terraform.InputValue, tfdiags.Diagnostics) {
	cache.configLock.RLock()
	defer cache.configLock.RUnlock()
	mp, exists := cache.ConfigVariables[mod.SourceDir]
	if !exists {
		return nil, nil
	}
	value, exists := mp[name]
	if !exists {
		return nil, nil
	}
	return value.Value, value.Diags
}

// SetConfigVariable sets a configuration variable in the context.
func (cache *VariableContext) SetConfigVariable(mod *configs.Module, name string, variable *configs.Variable) {
	var diags tfdiags.Diagnostics
	var value *terraform.InputValue
	if variable.Required() {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "No value for required variable",
			Detail: fmt.Sprintf("The module under test has a required variable %q with no set value. Use a -var or -var-file command line argument or add this variable into a \"variables\" block within the test file or run block.",
				variable.Name),
			Subject: variable.DeclRange.Ptr(),
		})

		value = &terraform.InputValue{
			Value:       cty.DynamicVal,
			SourceType:  terraform.ValueFromConfig,
			SourceRange: tfdiags.SourceRangeFromHCL(variable.DeclRange),
		}
	} else {
		value = &terraform.InputValue{
			Value:       cty.NilVal,
			SourceType:  terraform.ValueFromConfig,
			SourceRange: tfdiags.SourceRangeFromHCL(variable.DeclRange),
		}
	}
	cache.configLock.Lock()
	defer cache.configLock.Unlock()
	mp, exists := cache.ConfigVariables[mod.SourceDir]
	if !exists {
		mp = make(map[string]VariableWithDiag)
		cache.ConfigVariables[mod.SourceDir] = mp
	}
	mp[name] = VariableWithDiag{
		Value: value,
		Diags: diags,
	}
}

// GetRunOutput retrieves the output of a run from the context.
func (cache *VariableContext) GetRunOutput(run addrs.Run) (cty.Value, bool) {
	cache.outputLock.RLock()
	defer cache.outputLock.RUnlock()
	value, exists := cache.RunOutputs[run]
	return value, exists
}

// SetRunOutput sets the output of a run in the context.
func (cache *VariableContext) SetRunOutput(run addrs.Run, value cty.Value) {
	cache.outputLock.Lock()
	defer cache.outputLock.Unlock()
	cache.RunOutputs[run] = value
}
