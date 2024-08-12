// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"log"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// ValidateOpts are the various options the affect the details of how Terraform
// will validate a configuration.
type ValidateOpts struct {
	// ExternalProviders are clients for pre-configured providers that are
	// treated as being passed into the root module from the caller. This
	// is equivalent to writing a "providers" argument inside a "module"
	// block in the Terraform language, but for the root module the caller
	// is written in Go rather than the Terraform language.
	//
	// Note, that while Terraform Core will not call ValidateProviderConfig or
	// ConfigureProvider on any providers in this map, as with the other context
	// functions, the Validate function never calls ConfigureProvider anyway.
	//
	// Normally, the validate function would call the ValidateProviderConfig
	// function on the provider, but the config may rely on variables that are
	// not available to this function. Therefore, it is the responsibility of
	// the caller to ensure that the provider configurations are valid.
	ExternalProviders map[addrs.RootProviderConfig]providers.Interface
}

// Validate performs semantic validation of a configuration, and returns
// any warnings or errors.
//
// Syntax and structural checks are performed by the configuration loader,
// and so are not repeated here.
//
// Validate considers only the configuration and so it won't catch any
// errors caused by current values in the state, or other external information
// such as root module input variables. However, the Plan function includes
// all of the same checks as Validate, in addition to the other work it does
// to consider the previous run state and the planning options.
//
// The opts can be nil, and the ExternalProviders field of the opts can be nil.
func (c *Context) Validate(config *configs.Config, opts *ValidateOpts) tfdiags.Diagnostics {
	defer c.acquireRun("validate")()

	var diags tfdiags.Diagnostics

	if opts == nil {
		// Just make sure we don't get any nil pointer exceptions later.
		opts = &ValidateOpts{}
	}

	moreDiags := c.checkConfigDependencies(config)
	diags = diags.Append(moreDiags)
	// If required dependencies are not available then we'll bail early since
	// otherwise we're likely to just see a bunch of other errors related to
	// incompatibilities, which could be overwhelming for the user.
	if diags.HasErrors() {
		return diags
	}

	// There are some validation checks that happen when loading the provider
	// schemas, and we can catch them early to ensure we are in a position to
	// handle any errors.
	_, moreDiags = c.Schemas(config, nil)
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		return diags
	}

	log.Printf("[DEBUG] Building and walking validate graph")

	// Validate is to check if the given module is valid regardless of
	// input values, current state, etc. Therefore we populate all of the
	// input values with unknown values of the expected type, allowing us
	// to perform a type check without assuming any particular values.
	varValues := make(InputValues)
	for name, variable := range config.Module.Variables {
		ty := variable.Type
		if ty == cty.NilType {
			// Can't predict the type at all, so we'll just mark it as
			// cty.DynamicVal (unknown value of cty.DynamicPseudoType).
			ty = cty.DynamicPseudoType
		}
		varValues[name] = &InputValue{
			Value:      cty.UnknownVal(ty),
			SourceType: ValueFromUnknown,
		}
	}

	graph, moreDiags := (&PlanGraphBuilder{
		Config:                  config,
		Plugins:                 c.plugins,
		State:                   states.NewState(),
		RootVariableValues:      varValues,
		Operation:               walkValidate,
		ExternalProviderConfigs: opts.ExternalProviders,
		ImportTargets:           c.findImportTargets(config),
	}).Build(addrs.RootModuleInstance)
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		return diags
	}

	walker, walkDiags := c.walk(graph, walkValidate, &graphWalkOpts{
		Config:                  config,
		ProviderFuncResults:     providers.NewFunctionResultsTable(nil),
		ExternalProviderConfigs: opts.ExternalProviders,
	})
	diags = diags.Append(walker.NonFatalDiagnostics)
	diags = diags.Append(walkDiags)
	if walkDiags.HasErrors() {
		return diags
	}

	return diags
}
