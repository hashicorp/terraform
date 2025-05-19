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
	"github.com/hashicorp/terraform/internal/lang/langrefs"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// VariableCaches contains a mapping between test run blocks and evaluated
// variables. This is used to cache the results of evaluating variables so that
// they are only evaluated once per run.
//
// Each run block has its own configuration and therefore its own set of
// evaluated variables.
type VariableCaches struct {
	GlobalVariables map[string]backendrun.UnparsedVariableValue
	FileVariables   map[string]hcl.Expression

	caches    map[string]*VariableCache
	cacheLock sync.Mutex
}

func NewVariableCaches(opts ...func(*VariableCaches)) *VariableCaches {
	ret := &VariableCaches{
		GlobalVariables: make(map[string]backendrun.UnparsedVariableValue),
		FileVariables:   make(map[string]hcl.Expression),
		caches:          make(map[string]*VariableCache),
		cacheLock:       sync.Mutex{},
	}

	for _, opt := range opts {
		opt(ret)
	}

	return ret
}

// VariableCache contains the cache for a single run block. This cache contains
// the evaluated values for global and file-level variables.
type VariableCache struct {
	config *configs.Config

	globals terraform.InputValues
	files   terraform.InputValues

	values *VariableCaches // back reference so we can access the stored values
}

// GetCache returns the cache for the named run. If the cache does not exist, it
// is created and returned.
func (caches *VariableCaches) GetCache(name string, config *configs.Config) *VariableCache {
	caches.cacheLock.Lock()
	defer caches.cacheLock.Unlock()
	cache, exists := caches.caches[name]
	if !exists {
		cache = &VariableCache{
			config:  config,
			globals: make(terraform.InputValues),
			files:   make(terraform.InputValues),
			values:  caches,
		}
		caches.caches[name] = cache
	}
	return cache
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
func (cache *VariableCache) GetGlobalVariable(name string) (*terraform.InputValue, tfdiags.Diagnostics) {
	val, exists := cache.globals[name]
	if exists {
		return val, nil
	}

	variable, exists := cache.values.GlobalVariables[name]
	if !exists {
		return nil, nil
	}

	// TODO: We should also introduce a way to specify the mode in the test
	//   file itself. Suggestion, optional variable blocks.
	parsingMode := configs.VariableParseHCL

	if cfg, exists := cache.config.Module.Variables[name]; exists {
		parsingMode = cfg.ParsingMode
	}

	value, diags := variable.ParseVariableValue(parsingMode)
	if diags.HasErrors() {
		// In this case, the variable exists but we couldn't parse it. We'll
		// return a usable value so that we don't compound errors later by
		// claiming a variable doesn't exist when it does. We also return the
		// diagnostics explaining the error which will be shown to the user.
		value = &terraform.InputValue{
			Value: cty.DynamicVal,
		}
	}

	cache.globals[name] = value
	return value, diags
}

// GetFileVariable returns a value for the named file-level variable evaluated
// against the current run.
//
// This function caches the result of evaluating the variable so that it is
// only evaluated once per run.
//
// This function will return a valid input value if parsing fails for any reason
// so the caller can continue processing the configuration. The diagnostics
// returned will contain the error message that occurred during parsing and as
// such should be shown to the user.
func (cache *VariableCache) GetFileVariable(name string) (*terraform.InputValue, tfdiags.Diagnostics) {
	val, exists := cache.files[name]
	if exists {
		return val, nil
	}

	expr, exists := cache.values.FileVariables[name]
	if !exists {
		return nil, nil
	}

	var diags tfdiags.Diagnostics

	availableVariables := make(map[string]cty.Value)
	refs, refDiags := langrefs.ReferencesInExpr(addrs.ParseRefFromTestingScope, expr)
	for _, ref := range refs {
		if input, ok := ref.Subject.(addrs.InputVariable); ok {
			variable, variableDiags := cache.GetGlobalVariable(input.Name)
			diags = diags.Append(variableDiags)
			if variable != nil {
				availableVariables[input.Name] = variable.Value
			}
		}
	}
	diags = diags.Append(refDiags)

	if diags.HasErrors() {
		// There's no point trying to evaluate the variable as we know it will
		// fail. We'll just return a usable value so that we don't compound
		// errors later by claiming a variable doesn't exist when it does. We
		// also return the diagnostics explaining the error which will be shown
		// to the user.
		cache.files[name] = &terraform.InputValue{
			Value: cty.DynamicVal,
		}
		return cache.files[name], diags
	}

	ctx, ctxDiags := EvalContext(TargetFileVariable, map[string]hcl.Expression{name: expr}, availableVariables, nil)
	diags = diags.Append(ctxDiags)

	if ctxDiags.HasErrors() {
		// If we couldn't build the context, we won't actually process these
		// variables. Instead, we'll fill them with an empty value but still
		// make a note that the user did provide them.
		cache.files[name] = &terraform.InputValue{
			Value: cty.DynamicVal,
		}
		return cache.files[name], diags
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

	cache.files[name] = &terraform.InputValue{
		Value:       value,
		SourceType:  terraform.ValueFromConfig,
		SourceRange: tfdiags.SourceRangeFromHCL(expr.Range()),
	}
	return cache.files[name], diags
}
