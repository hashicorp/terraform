// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"log"

	"github.com/zclconf/go-cty/cty"

	tfaddr "github.com/hashicorp/terraform-registry-address"
	"github.com/hashicorp/terraform/internal/providers"
)

// Provider is an implementation of providers.Interface
type Provider struct{}

// NewProvider returns a new terraform provider
func NewProvider() providers.Interface {
	return &Provider{}
}

// GetSchema returns the complete schema for the provider.
func (p *Provider) GetProviderSchema() providers.GetProviderSchemaResponse {
	resp := providers.GetProviderSchemaResponse{
		ServerCapabilities: providers.ServerCapabilities{
			MoveResourceState: true,
		},
		DataSources: map[string]providers.Schema{
			"terraform_remote_state": dataSourceRemoteStateGetSchema(),
		},
		ResourceTypes: map[string]providers.Schema{
			"terraform_data": dataStoreResourceSchema(),
		},
		Functions: map[string]providers.FunctionDecl{
			"encode_tfvars": {
				Summary:     "Produce a string representation of an object using the same syntax as for `.tfvars` files",
				Description: "A rarely-needed function which takes an object value and produces a string containing a description of that object using the same syntax as Terraform CLI would expect in a `.tfvars`.",
				Parameters: []providers.FunctionParam{
					{
						Name:               "value",
						Type:               cty.DynamicPseudoType,
						AllowUnknownValues: true, // to perform refinements
					},
				},
				ReturnType: cty.String,
			},
			"decode_tfvars": {
				Summary:     "Parse a string containing syntax like that used in a `.tfvars` file",
				Description: "A rarely-needed function which takes a string containing the content of a `.tfvars` file and returns an object describing the raw variable values it defines.",
				Parameters: []providers.FunctionParam{
					{
						Name: "src",
						Type: cty.String,
					},
				},
				ReturnType: cty.DynamicPseudoType,
			},
			"encode_expr": {
				Summary:     "Produce a string representation of an arbitrary value using Terraform expression syntax",
				Description: "A rarely-needed function which takes any value and produces a string containing Terraform language expression syntax approximating that value.",
				Parameters: []providers.FunctionParam{
					{
						Name:               "value",
						Type:               cty.DynamicPseudoType,
						AllowUnknownValues: true, // to perform refinements
					},
				},
				ReturnType: cty.String,
			},
		},
	}
	providers.SchemaCache.Set(tfaddr.NewProvider(tfaddr.BuiltInProviderHost, tfaddr.BuiltInProviderNamespace, "terraform"), resp)
	return resp
}

// ValidateProviderConfig is used to validate the configuration values.
func (p *Provider) ValidateProviderConfig(req providers.ValidateProviderConfigRequest) providers.ValidateProviderConfigResponse {
	// At this moment there is nothing to configure for the terraform provider,
	// so we will happily return without taking any action
	var res providers.ValidateProviderConfigResponse
	res.PreparedConfig = req.Config
	return res
}

// ValidateDataResourceConfig is used to validate the data source configuration values.
func (p *Provider) ValidateDataResourceConfig(req providers.ValidateDataResourceConfigRequest) providers.ValidateDataResourceConfigResponse {
	// FIXME: move the backend configuration validate call that's currently
	// inside the read method  into here so that we can catch provider configuration
	// errors in terraform validate as well as during terraform plan.
	var res providers.ValidateDataResourceConfigResponse

	// This should not happen
	if req.TypeName != "terraform_remote_state" {
		res.Diagnostics.Append(fmt.Errorf("Error: unsupported data source %s", req.TypeName))
		return res
	}

	diags := dataSourceRemoteStateValidate(req.Config)
	res.Diagnostics = diags

	return res
}

// Configure configures and initializes the provider.
func (p *Provider) ConfigureProvider(providers.ConfigureProviderRequest) providers.ConfigureProviderResponse {
	// At this moment there is nothing to configure for the terraform provider,
	// so we will happily return without taking any action
	var res providers.ConfigureProviderResponse
	return res
}

// ReadDataSource returns the data source's current state.
func (p *Provider) ReadDataSource(req providers.ReadDataSourceRequest) providers.ReadDataSourceResponse {
	// call function
	var res providers.ReadDataSourceResponse

	// This should not happen
	if req.TypeName != "terraform_remote_state" {
		res.Diagnostics.Append(fmt.Errorf("Error: unsupported data source %s", req.TypeName))
		return res
	}

	newState, diags := dataSourceRemoteStateRead(req.Config)

	res.State = newState
	res.Diagnostics = diags

	return res
}

// Stop is called when the provider should halt any in-flight actions.
func (p *Provider) Stop() error {
	log.Println("[DEBUG] terraform provider cannot Stop")
	return nil
}

// All the Resource-specific functions are below.
// The terraform provider supplies a single data source, `terraform_remote_state`
// and no resources.

// UpgradeResourceState is called when the state loader encounters an
// instance state whose schema version is less than the one reported by the
// currently-used version of the corresponding provider, and the upgraded
// result is used for any further processing.
func (p *Provider) UpgradeResourceState(req providers.UpgradeResourceStateRequest) providers.UpgradeResourceStateResponse {
	return upgradeDataStoreResourceState(req)
}

// ReadResource refreshes a resource and returns its current state.
func (p *Provider) ReadResource(req providers.ReadResourceRequest) providers.ReadResourceResponse {
	return readDataStoreResourceState(req)
}

// PlanResourceChange takes the current state and proposed state of a
// resource, and returns the planned final state.
func (p *Provider) PlanResourceChange(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
	return planDataStoreResourceChange(req)
}

// ApplyResourceChange takes the planned state for a resource, which may
// yet contain unknown computed values, and applies the changes returning
// the final state.
func (p *Provider) ApplyResourceChange(req providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
	return applyDataStoreResourceChange(req)
}

// ImportResourceState requests that the given resource be imported.
func (p *Provider) ImportResourceState(req providers.ImportResourceStateRequest) providers.ImportResourceStateResponse {
	if req.TypeName == "terraform_data" {
		return importDataStore(req)
	}

	panic("unimplemented - terraform_remote_state has no resources")
}

// MoveResourceState requests that the given resource be moved.
func (p *Provider) MoveResourceState(req providers.MoveResourceStateRequest) providers.MoveResourceStateResponse {
	switch req.TargetTypeName {
	case "terraform_data":
		return moveDataStoreResourceState(req)
	default:
		var resp providers.MoveResourceStateResponse

		resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("Error: unsupported resource %s", req.TargetTypeName))

		return resp
	}
}

// ValidateResourceConfig is used to to validate the resource configuration values.
func (p *Provider) ValidateResourceConfig(req providers.ValidateResourceConfigRequest) providers.ValidateResourceConfigResponse {
	return validateDataStoreResourceConfig(req)
}

// CallFunction would call a function contributed by this provider, but this
// provider has no functions and so this function just panics.
func (p *Provider) CallFunction(req providers.CallFunctionRequest) providers.CallFunctionResponse {
	fn, ok := functions[req.FunctionName]
	if !ok {
		// Should not get here if the caller is behaving correctly, because
		// we don't declare any functions in our schema that we don't have
		// implementations for.
		return providers.CallFunctionResponse{
			Err: fmt.Errorf("provider has no function named %q", req.FunctionName),
		}
	}

	// NOTE: We assume that none of the arguments can be marked, because we're
	// expecting to be called from logic in Terraform Core that strips marks
	// before calling a provider-contributed function, and then reapplies them
	// afterwards.

	result, err := fn(req.Arguments)
	if err != nil {
		return providers.CallFunctionResponse{
			Err: err,
		}
	}
	return providers.CallFunctionResponse{
		Result: result,
	}
}

// Close is a noop for this provider, since it's run in-process.
func (p *Provider) Close() error {
	return nil
}
