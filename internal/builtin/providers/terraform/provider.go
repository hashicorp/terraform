// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"bytes"
	"fmt"
	"log"
	"os"

	tfaddr "github.com/hashicorp/terraform-registry-address"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states/statefile"
)

// Provider is an implementation of providers.Interface
type Provider struct {

	// State storage implementation(s)
	inMem *InMemStoreSingle // terraform_inmem
}

var _ providers.Interface = &Provider{}

// NewProvider returns a new terraform provider
func NewProvider() providers.Interface {
	return &Provider{
		inMem: &InMemStoreSingle{},
	}
}

// NewProvider returns a new terraform provider where the internal
// state store(s) all have the default workspace already existing
func NewProviderWithDefaultState() providers.Interface {
	// Get the empty state file as bytes
	f := statefile.New(nil, "", 0)

	var buf bytes.Buffer
	err := statefile.Write(f, &buf)
	if err != nil {
		panic(err)
	}
	emptyStateBytes := buf.Bytes()

	// Return a provider where all state stores have existing default workspaces
	return &Provider{
		inMem: &InMemStoreSingle{
			states: stateMap{
				m: map[string][]byte{
					backend.DefaultStateName: emptyStateBytes,
				},
			},
		},
	}
}

// GetSchema returns the complete schema for the provider.
func (p *Provider) GetProviderSchema() providers.GetProviderSchemaResponse {
	resp := providers.GetProviderSchemaResponse{
		Provider: providers.Schema{},
		ServerCapabilities: providers.ServerCapabilities{
			MoveResourceState: true,
		},
		DataSources: map[string]providers.Schema{
			"terraform_remote_state": dataSourceRemoteStateGetSchema(),
		},
		ResourceTypes: map[string]providers.Schema{
			"terraform_data": dataStoreResourceSchema(),
		},
		EphemeralResourceTypes: map[string]providers.Schema{},
		ListResourceTypes:      map[string]providers.Schema{},
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
		StateStores: map[string]providers.Schema{},
		Actions:     map[string]providers.ActionSchema{},
	}
	providers.SchemaCache.Set(tfaddr.NewProvider(tfaddr.BuiltInProviderHost, tfaddr.BuiltInProviderNamespace, "terraform"), resp)

	// Only include the inmem state store in the provider when `TF_ACC` is set in the environment
	// Excluding this from the schemas is sufficient to block usage.
	if v := os.Getenv("TF_ACC"); v != "" {
		resp.StateStores[inMemStoreName] = stateStoreInMemGetSchema()
	}

	return resp
}

func (p *Provider) GetResourceIdentitySchemas() providers.GetResourceIdentitySchemasResponse {
	return providers.GetResourceIdentitySchemasResponse{
		IdentityTypes: map[string]providers.IdentitySchema{
			"terraform_data": dataStoreResourceIdentitySchema(),
		},
	}
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
		res.Diagnostics = res.Diagnostics.Append(fmt.Errorf("Error: unsupported data source %s", req.TypeName))
		return res
	}

	diags := dataSourceRemoteStateValidate(req.Config)
	res.Diagnostics = diags

	return res
}

func (p *Provider) ValidateListResourceConfig(req providers.ValidateListResourceConfigRequest) providers.ValidateListResourceConfigResponse {
	var resp providers.ValidateListResourceConfigResponse
	resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("unsupported list resource type %q", req.TypeName))
	return resp
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
		res.Diagnostics = res.Diagnostics.Append(fmt.Errorf("Error: unsupported data source %s", req.TypeName))
		return res
	}

	newState, diags := dataSourceRemoteStateRead(req.Config)

	res.State = newState
	res.Diagnostics = diags

	return res
}

func (p *Provider) GenerateResourceConfig(providers.GenerateResourceConfigRequest) providers.GenerateResourceConfigResponse {
	panic("not implemented")
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

func (p *Provider) UpgradeResourceIdentity(req providers.UpgradeResourceIdentityRequest) providers.UpgradeResourceIdentityResponse {
	return upgradeDataStoreResourceIdentity(req)
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

	panic("unimplemented: cannot import resource type " + req.TypeName)
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

func (p *Provider) ValidateEphemeralResourceConfig(req providers.ValidateEphemeralResourceConfigRequest) providers.ValidateEphemeralResourceConfigResponse {
	var resp providers.ValidateEphemeralResourceConfigResponse
	resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("unsupported ephemeral resource type %q", req.TypeName))
	return resp
}

// OpenEphemeralResource implements providers.Interface.
func (p *Provider) OpenEphemeralResource(req providers.OpenEphemeralResourceRequest) providers.OpenEphemeralResourceResponse {
	var resp providers.OpenEphemeralResourceResponse
	resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("unsupported ephemeral resource type %q", req.TypeName))
	return resp
}

// RenewEphemeralResource implements providers.Interface.
func (p *Provider) RenewEphemeralResource(req providers.RenewEphemeralResourceRequest) providers.RenewEphemeralResourceResponse {
	var resp providers.RenewEphemeralResourceResponse
	resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("unsupported ephemeral resource type %q", req.TypeName))
	return resp
}

// CloseEphemeralResource implements providers.Interface.
func (p *Provider) CloseEphemeralResource(req providers.CloseEphemeralResourceRequest) providers.CloseEphemeralResourceResponse {
	var resp providers.CloseEphemeralResourceResponse
	resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("unsupported ephemeral resource type %q", req.TypeName))
	return resp
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

func (p *Provider) ListResource(req providers.ListResourceRequest) providers.ListResourceResponse {
	var resp providers.ListResourceResponse
	resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("unsupported list resource type %q", req.TypeName))
	return resp
}

func (p *Provider) ValidateStateStoreConfig(req providers.ValidateStateStoreConfigRequest) providers.ValidateStateStoreConfigResponse {
	if req.TypeName == inMemStoreName {
		return p.inMem.ValidateStateStoreConfig(req)
	}

	var resp providers.ValidateStateStoreConfigResponse
	resp.Diagnostics.Append(fmt.Errorf("unsupported state store type %q", req.TypeName))
	return resp
}

func (p *Provider) ConfigureStateStore(req providers.ConfigureStateStoreRequest) providers.ConfigureStateStoreResponse {
	if req.TypeName == inMemStoreName {
		return p.inMem.ConfigureStateStore(req)
	}

	var resp providers.ConfigureStateStoreResponse
	resp.Diagnostics.Append(fmt.Errorf("unsupported state store type %q", req.TypeName))
	return resp
}

func (p *Provider) ReadStateBytes(req providers.ReadStateBytesRequest) providers.ReadStateBytesResponse {
	if req.TypeName == inMemStoreName {
		return p.inMem.ReadStateBytes(req)
	}

	var resp providers.ReadStateBytesResponse
	resp.Diagnostics.Append(fmt.Errorf("unsupported state store type %q", req.TypeName))
	return resp
}

func (p *Provider) WriteStateBytes(req providers.WriteStateBytesRequest) providers.WriteStateBytesResponse {
	if req.TypeName == inMemStoreName {
		return p.inMem.WriteStateBytes(req)
	}

	var resp providers.WriteStateBytesResponse
	resp.Diagnostics.Append(fmt.Errorf("unsupported state store type %q", req.TypeName))
	return resp
}

func (p *Provider) LockState(req providers.LockStateRequest) providers.LockStateResponse {
	if req.TypeName == inMemStoreName {
		return p.inMem.LockState(req)
	}

	var resp providers.LockStateResponse
	resp.Diagnostics.Append(fmt.Errorf("unsupported state store type %q", req.TypeName))
	return resp
}

func (p *Provider) UnlockState(req providers.UnlockStateRequest) providers.UnlockStateResponse {
	if req.TypeName == inMemStoreName {
		return p.inMem.UnlockState(req)
	}

	var resp providers.UnlockStateResponse
	resp.Diagnostics.Append(fmt.Errorf("unsupported state store type %q", req.TypeName))
	return resp
}

func (p *Provider) GetStates(req providers.GetStatesRequest) providers.GetStatesResponse {
	if req.TypeName == inMemStoreName {
		return p.inMem.GetStates(req)
	}

	var resp providers.GetStatesResponse
	resp.Diagnostics.Append(fmt.Errorf("unsupported state store type %q", req.TypeName))
	return resp
}

func (p *Provider) DeleteState(req providers.DeleteStateRequest) providers.DeleteStateResponse {
	if req.TypeName == inMemStoreName {
		return p.inMem.DeleteState(req)
	}

	var resp providers.DeleteStateResponse
	resp.Diagnostics.Append(fmt.Errorf("unsupported state store type %q", req.TypeName))
	return resp
}

func (p *Provider) PlanAction(req providers.PlanActionRequest) providers.PlanActionResponse {
	var resp providers.PlanActionResponse

	switch req.ActionType {
	default:
		resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("unsupported action %q", req.ActionType))
	}

	return resp
}

func (p *Provider) InvokeAction(req providers.InvokeActionRequest) providers.InvokeActionResponse {
	var resp providers.InvokeActionResponse

	switch req.ActionType {
	default:
		resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("unsupported action %q", req.ActionType))
	}

	return resp
}

func (p *Provider) ValidateActionConfig(req providers.ValidateActionConfigRequest) providers.ValidateActionConfigResponse {
	var resp providers.ValidateActionConfigResponse

	switch req.TypeName {
	default:
		resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("unsupported action %q", req.TypeName))
	}

	return resp
}

// Close is a noop for this provider, since it's run in-process.
func (p *Provider) Close() error {
	return nil
}
