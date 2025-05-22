// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

// simple provider a minimal provider implementation for testing
package simple

import (
	"errors"
	"time"

	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"

	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
)

type simple struct {
	schema providers.GetProviderSchemaResponse
}

func Provider() providers.Interface {
	simpleResource := providers.Schema{
		Body: &configschema.Block{
			Attributes: map[string]*configschema.Attribute{
				"id": {
					Computed: true,
					Type:     cty.String,
				},
				"value": {
					Optional: true,
					Type:     cty.String,
				},
			},
		},
	}

	return simple{
		schema: providers.GetProviderSchemaResponse{
			Provider: providers.Schema{
				Body: nil,
			},
			ResourceTypes: map[string]providers.Schema{
				"simple_resource": simpleResource,
			},
			DataSources: map[string]providers.Schema{
				"simple_resource": simpleResource,
			},
			EphemeralResourceTypes: map[string]providers.Schema{
				"simple_resource": simpleResource,
			},
			ServerCapabilities: providers.ServerCapabilities{
				PlanDestroy: true,
			},
		},
	}
}

func (s simple) GetProviderSchema() providers.GetProviderSchemaResponse {
	return s.schema
}

func (s simple) GetResourceIdentitySchemas() providers.GetResourceIdentitySchemasResponse {
	return providers.GetResourceIdentitySchemasResponse{
		IdentityTypes: map[string]providers.IdentitySchema{
			"simple_resource": {
				Version: 0,
				Body: &configschema.Object{
					Attributes: map[string]*configschema.Attribute{
						"id": {
							Type:     cty.String,
							Required: true,
						},
					},
					Nesting: configschema.NestingSingle,
				},
			},
		},
	}
}

func (s simple) ValidateProviderConfig(req providers.ValidateProviderConfigRequest) (resp providers.ValidateProviderConfigResponse) {
	return resp
}

func (s simple) ValidateResourceConfig(req providers.ValidateResourceConfigRequest) (resp providers.ValidateResourceConfigResponse) {
	return resp
}

func (s simple) ValidateDataResourceConfig(req providers.ValidateDataResourceConfigRequest) (resp providers.ValidateDataResourceConfigResponse) {
	return resp
}

func (s simple) ValidateListResourceConfig(req providers.ValidateListResourceConfigRequest) (resp providers.ValidateListResourceConfigResponse) {
	return resp
}

func (p simple) UpgradeResourceState(req providers.UpgradeResourceStateRequest) (resp providers.UpgradeResourceStateResponse) {
	ty := p.schema.ResourceTypes[req.TypeName].Body.ImpliedType()
	val, err := ctyjson.Unmarshal(req.RawStateJSON, ty)
	resp.Diagnostics = resp.Diagnostics.Append(err)
	resp.UpgradedState = val
	return resp
}

func (p simple) UpgradeResourceIdentity(req providers.UpgradeResourceIdentityRequest) (resp providers.UpgradeResourceIdentityResponse) {
	schema := p.GetResourceIdentitySchemas().IdentityTypes[req.TypeName].Body
	ty := schema.ImpliedType()
	val, err := ctyjson.Unmarshal(req.RawIdentityJSON, ty)
	resp.Diagnostics = resp.Diagnostics.Append(err)
	resp.UpgradedIdentity = val
	return resp
}

func (s simple) ConfigureProvider(providers.ConfigureProviderRequest) (resp providers.ConfigureProviderResponse) {
	return resp
}

func (s simple) Stop() error {
	return nil
}

func (s simple) ReadResource(req providers.ReadResourceRequest) (resp providers.ReadResourceResponse) {
	// just return the same state we received
	resp.NewState = req.PriorState
	resp.Identity = req.CurrentIdentity
	return resp
}

func (s simple) PlanResourceChange(req providers.PlanResourceChangeRequest) (resp providers.PlanResourceChangeResponse) {
	if req.ProposedNewState.IsNull() {
		// destroy op
		resp.PlannedState = req.ProposedNewState
		resp.PlannedPrivate = req.PriorPrivate
		return resp
	}

	m := req.ProposedNewState.AsValueMap()
	_, ok := m["id"]
	if !ok {
		m["id"] = cty.UnknownVal(cty.String)
	}

	resp.PlannedState = cty.ObjectVal(m)
	return resp
}

func (s simple) ApplyResourceChange(req providers.ApplyResourceChangeRequest) (resp providers.ApplyResourceChangeResponse) {
	if req.PlannedState.IsNull() {
		resp.NewState = req.PlannedState
		return resp
	}

	m := req.PlannedState.AsValueMap()
	_, ok := m["id"]
	if !ok {
		m["id"] = cty.StringVal(time.Now().String())
	}
	resp.NewState = cty.ObjectVal(m)
	resp.NewIdentity = req.PlannedIdentity

	return resp
}

func (s simple) ImportResourceState(providers.ImportResourceStateRequest) (resp providers.ImportResourceStateResponse) {
	resp.Diagnostics = resp.Diagnostics.Append(errors.New("unsupported"))
	return resp
}

func (s simple) MoveResourceState(providers.MoveResourceStateRequest) (resp providers.MoveResourceStateResponse) {
	// We don't expose the move_resource_state capability, so this should never
	// be called.
	resp.Diagnostics = resp.Diagnostics.Append(errors.New("unsupported"))
	return resp
}

func (s simple) ReadDataSource(req providers.ReadDataSourceRequest) (resp providers.ReadDataSourceResponse) {
	m := req.Config.AsValueMap()
	m["id"] = cty.StringVal("static_id")
	resp.State = cty.ObjectVal(m)
	return resp
}

func (p simple) ValidateEphemeralResourceConfig(req providers.ValidateEphemeralResourceConfigRequest) providers.ValidateEphemeralResourceConfigResponse {
	// Our schema doesn't include any ephemeral resource types, so it should be
	// impossible to get in here.
	panic("ValidateEphemeralResourceConfig on provider that didn't declare any ephemeral resource types")
}

func (s simple) OpenEphemeralResource(providers.OpenEphemeralResourceRequest) providers.OpenEphemeralResourceResponse {
	// Our schema doesn't include any ephemeral resource types, so it should be
	// impossible to get in here.
	panic("OpenEphemeralResource on provider that didn't declare any ephemeral resource types")
}

func (s simple) RenewEphemeralResource(providers.RenewEphemeralResourceRequest) providers.RenewEphemeralResourceResponse {
	// Our schema doesn't include any ephemeral resource types, so it should be
	// impossible to get in here.
	panic("RenewEphemeralResource on provider that didn't declare any ephemeral resource types")
}

func (s simple) CloseEphemeralResource(providers.CloseEphemeralResourceRequest) providers.CloseEphemeralResourceResponse {
	// Our schema doesn't include any ephemeral resource types, so it should be
	// impossible to get in here.
	panic("CloseEphemeralResource on provider that didn't declare any ephemeral resource types")
}

func (s simple) CallFunction(req providers.CallFunctionRequest) (resp providers.CallFunctionResponse) {
	// Our schema doesn't include any functions, so it should be impossible
	// to get in here.
	panic("CallFunction on provider that didn't declare any functions")
}

func (s simple) Close() error {
	return nil
}
