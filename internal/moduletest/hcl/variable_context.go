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
	globalLock            sync.RWMutex

	// ParsedFileVariables contains the evaluated values for file-level variables.
	ParsedFileVariables terraform.InputValues
	fileLock            sync.RWMutex

	// ConfigVariables contains the evaluated values for variables declared in the
	// configuration.
	// Each key is the source directory of the module that the variables are
	// declared in.
	ConfigVariables map[string]terraform.InputValues
	configLock      sync.RWMutex

	// RunVariables contains the evaluated values for variables declared in the
	// run blocks.
	RunVariables map[string]terraform.InputValues
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
}

// NewTestContext creates a new VariableContext with the provided configuration and variables.
func NewTestContext(config *configs.Config, globalVariables map[string]backendrun.UnparsedVariableValue, fileVariables map[string]hcl.Expression, runOutputs map[addrs.Run]cty.Value) *VariableContext {
	return &VariableContext{
		Config:                config,
		GlobalVariables:       globalVariables,
		FileVariables:         fileVariables,
		ParsedGlobalVariables: make(terraform.InputValues),
		ParsedFileVariables:   make(terraform.InputValues),
		ConfigVariables:       make(map[string]terraform.InputValues),
		RunVariables:          make(map[string]terraform.InputValues),
		RunOutputs:            runOutputs,
	}
}

// SetGlobalVariable sets a global variable in the context.
func (cache *VariableContext) SetGlobalVariable(name string, value *terraform.InputValue) {
	cache.globalLock.Lock()
	defer cache.globalLock.Unlock()
	cache.ParsedGlobalVariables[name] = value
}

// SetFileVariable sets a file-level variable in the context.
func (cache *VariableContext) SetFileVariable(name string, value *terraform.InputValue) {
	cache.fileLock.Lock()
	defer cache.fileLock.Unlock()
	cache.ParsedFileVariables[name] = value
}

// SetRunVariable sets a run-level variable in the context.
func (cache *VariableContext) SetRunVariable(runName, varName string, value *terraform.InputValue) {
	cache.runLock.Lock()
	defer cache.runLock.Unlock()
	store, exists := cache.RunVariables[runName]
	if !exists {
		store = make(terraform.InputValues)
		cache.RunVariables[runName] = store
	}
	store[varName] = value
}

// GetGlobalVariable retrieves a global variable from the context.
func (cache *VariableContext) GetGlobalVariable(name string) (*terraform.InputValue, error) {
	cache.globalLock.RLock()
	defer cache.globalLock.RUnlock()
	value, exists := cache.ParsedGlobalVariables[name]
	if !exists {
		return nil, nil
	}
	return value, nil
}

// GetFileVariable retrieves a file-level variable from the context.
func (cache *VariableContext) GetFileVariable(name string) (*terraform.InputValue, error) {
	cache.fileLock.RLock()
	defer cache.fileLock.RUnlock()
	value, exists := cache.ParsedFileVariables[name]
	if !exists {
		return nil, nil
	}
	return value, nil
}

// GetRunVariable retrieves a run-level variable from the context.
func (cache *VariableContext) GetRunVariable(runName, varName string) (*terraform.InputValue, error) {
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
	return value, nil
}

// GetParsedVariables retrieves all parsed variables for a given key and run name.
func (cache *VariableContext) GetParsedVariables(mod *configs.Module, runName string) terraform.InputValues {
	variables := make(terraform.InputValues)
	cache.configLock.RLock()
	if configVariables, exists := cache.ConfigVariables[mod.SourceDir]; exists {
		for name, value := range configVariables {
			variables[name] = value
		}
	}
	cache.configLock.RUnlock()

	cache.globalLock.RLock()
	for name, value := range cache.ParsedGlobalVariables {
		variables[name] = value
	}
	cache.globalLock.RUnlock()

	cache.fileLock.RLock()
	for name, value := range cache.ParsedFileVariables {
		variables[name] = value
	}
	cache.fileLock.RUnlock()

	cache.runLock.RLock()
	if runVariables, exists := cache.RunVariables[runName]; exists {
		for name, value := range runVariables {
			variables[name] = value
		}
	}
	cache.runLock.RUnlock()

	return variables
}

// GetConfigVariable retrieves a configuration variable from the context.
func (cache *VariableContext) GetConfigVariable(mod *configs.Module, name string) (*terraform.InputValue, error) {
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
	return value, nil
}

// SetConfigVariable sets a configuration variable in the context.
func (cache *VariableContext) SetConfigVariable(mod *configs.Module, name string, value *terraform.InputValue) {
	cache.configLock.Lock()
	defer cache.configLock.Unlock()
	mp, exists := cache.ConfigVariables[mod.SourceDir]
	if !exists {
		mp = make(terraform.InputValues)
		cache.ConfigVariables[mod.SourceDir] = mp
	}
	mp[name] = value
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
