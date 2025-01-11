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
	globalLock            sync.Mutex

	// ParsedFileVariables contains the evaluated values for file-level variables.
	ParsedFileVariables terraform.InputValues
	fileLock            sync.Mutex

	// ConfigVariables contains the evaluated values for variables declared in the
	// configuration.
	ConfigVariables map[string]terraform.InputValues

	// RunVariables contains the evaluated values for variables declared in the
	// run blocks.
	RunVariables map[string]terraform.InputValues

	// RunOutputs is a mapping from run addresses to cty object values
	// representing the collected output values from the module under test.
	//
	// This is used to allow run blocks to refer back to the output values of
	// previous run blocks. It is passed into the Evaluate functions that
	// validate the test assertions and is used when calculating values for
	// variables within run blocks.
	RunOutputs map[addrs.Run]cty.Value
}

func NewTestContext(config *configs.Config, globalVariables map[string]backendrun.UnparsedVariableValue, fileVariables map[string]hcl.Expression, runOutputs map[addrs.Run]cty.Value) *VariableContext {
	return &VariableContext{
		Config:                config,
		GlobalVariables:       globalVariables,
		FileVariables:         fileVariables,
		ParsedGlobalVariables: make(terraform.InputValues),
		globalLock:            sync.Mutex{},
		ParsedFileVariables:   make(terraform.InputValues),
		fileLock:              sync.Mutex{},
		ConfigVariables:       make(map[string]terraform.InputValues),
		RunVariables:          make(map[string]terraform.InputValues),
		RunOutputs:            runOutputs,
	}
}

func (cache *VariableContext) SetGlobalVariable(name string, value *terraform.InputValue) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	cache.ParsedGlobalVariables[name] = value
	return diags
}

func (cache *VariableContext) SetFileVariable(name string, value *terraform.InputValue) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	cache.ParsedFileVariables[name] = value
	return diags
}

func (cache *VariableContext) SetRunVariable(runName, varName string, value *terraform.InputValue) tfdiags.Diagnostics {
	store, exists := cache.RunVariables[runName]
	if !exists {
		store = make(terraform.InputValues)
		cache.RunVariables[runName] = store
	}
	cache.RunVariables[runName][varName] = value
	return nil
}

func (cache *VariableContext) GetGlobalVariable(name string) (*terraform.InputValue, tfdiags.Diagnostics) {
	cache.globalLock.Lock()
	defer cache.globalLock.Unlock()
	return cache.ParsedGlobalVariables[name], nil
}

func (cache *VariableContext) GetFileVariable(name string) (*terraform.InputValue, tfdiags.Diagnostics) {
	cache.fileLock.Lock()
	defer cache.fileLock.Unlock()
	return cache.ParsedFileVariables[name], nil
}

func (cache *VariableContext) GetRunVariable(runName, varName string) (*terraform.InputValue, tfdiags.Diagnostics) {
	store, exists := cache.RunVariables[runName]
	if !exists {
		return nil, nil
	}
	return store[varName], nil
}

func (cache *VariableContext) GetParsedVariables(key, runName string) terraform.InputValues {
	variables := make(terraform.InputValues)
	// The order of these assignments is important. The variables from the
	// config are the lowest priority and should be overridden by the variables
	// from the parsed files and global variables.
	if configVariables, exists := cache.ConfigVariables[key]; exists {
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

func (cache *VariableContext) GetConfigVariable(mod *configs.Module, name string) (*terraform.InputValue, tfdiags.Diagnostics) {
	mp, exists := cache.ConfigVariables[mod.SourceDir]
	if !exists {
		return nil, nil
	}
	value, exists := mp[name]
	if !exists {
		return nil, nil
	}

	return value, nil
}

func (cache *VariableContext) SetConfigVariable(mod *configs.Module, name string, value *terraform.InputValue) tfdiags.Diagnostics {
	mp, exists := cache.ConfigVariables[mod.SourceDir]
	if !exists {
		mp = make(terraform.InputValues)
		cache.ConfigVariables[mod.SourceDir] = mp
	}
	mp[name] = value
	return nil
}
