// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package hcl

import (
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
	FileVariables   map[string]hcl.Expression

	Config *configs.Config

	ParsedGlobalVariables terraform.InputValues
	ParsedFileVariables   terraform.InputValues
	ConfigVariables       map[string]terraform.InputValues
	RunVariables          map[string]terraform.InputValues

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

func (cache *TestContext) GetConfigVariable(mod *configs.Module, name string) (*terraform.InputValue, tfdiags.Diagnostics) {
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

func (cache *TestContext) SetConfigVariable(mod *configs.Module, name string, value *terraform.InputValue) tfdiags.Diagnostics {
	mp, exists := cache.ConfigVariables[mod.SourceDir]
	if !exists {
		mp = make(terraform.InputValues)
		cache.ConfigVariables[mod.SourceDir] = mp
	}
	mp[name] = value
	return nil
}
