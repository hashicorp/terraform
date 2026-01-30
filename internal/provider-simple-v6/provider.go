// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

// simple provider a minimal provider implementation for testing
package simple

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states/statefile"
)

type simple struct {
	schema providers.GetProviderSchemaResponse

	inMem *InMemStoreSingle
	fs    *FsStore
}

var _ providers.StateStoreChunkSizeSetter = &simple{}

// Provider returns an instance of providers.Interface
func Provider() providers.Interface {
	return provider()
}

// ProviderWithDefaultState returns an instance of providers.Interface,
// where the underlying simple struct has been changed to indicate that the
// 'default' state has already been created as an empty state file.
func ProviderWithDefaultState() providers.Interface {
	// Get the empty state file as bytes
	f := statefile.New(nil, "", 0)

	var buf bytes.Buffer
	err := statefile.Write(f, &buf)
	if err != nil {
		panic(err)
	}
	emptyStateBytes := buf.Bytes()

	p := provider()

	p.inMem.states.m = make(map[string][]byte, 1)
	p.inMem.states.m[backend.DefaultStateName] = emptyStateBytes

	return p
}

// provider returns an instance of simple
func provider() simple {
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
		Identity: &configschema.Object{
			Attributes: map[string]*configschema.Attribute{
				"id": {
					Type:     cty.String,
					Required: true,
				},
			},
			Nesting: configschema.NestingSingle,
		},
	}

	provider := simple{
		schema: providers.GetProviderSchemaResponse{
			Provider: providers.Schema{
				Body: &configschema.Block{
					Description: "This is terraform-provider-simple v6",
				},
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
			ListResourceTypes: map[string]providers.Schema{
				"simple_resource": {
					Body: &configschema.Block{
						Attributes: map[string]*configschema.Attribute{
							"value": {
								Optional: true,
								Type:     cty.String,
							},
						},
					},
				},
			},
			Actions: map[string]providers.ActionSchema{},
			StateStores: map[string]providers.Schema{
				inMemStoreName: stateStoreInMemGetSchema(), // simple6_inmem
				fsStoreName:    stateStoreFsGetSchema(),    // simple6_fs
			},
			ServerCapabilities: providers.ServerCapabilities{
				PlanDestroy:               true,
				GetProviderSchemaOptional: true,
			},
			Functions: map[string]providers.FunctionDecl{
				"noop": {
					Parameters: []providers.FunctionParam{
						{
							Name:               "noop",
							Type:               cty.DynamicPseudoType,
							AllowNullValue:     true,
							AllowUnknownValues: true,
							Description:        "any value",
							DescriptionKind:    configschema.StringPlain,
						},
					},
					ReturnType:      cty.DynamicPseudoType,
					Description:     "noop takes any single argument and returns the same value",
					DescriptionKind: configschema.StringPlain,
				},
			},
		},

		// the "default" state doesn't exist by default here; needs explicit creation via init command
		inMem: &InMemStoreSingle{},
		fs:    &FsStore{},
	}

	return provider
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

func (s simple) GenerateResourceConfig(req providers.GenerateResourceConfigRequest) (resp providers.GenerateResourceConfigResponse) {
	panic("not implemented")
}

func (s simple) PlanResourceChange(req providers.PlanResourceChangeRequest) (resp providers.PlanResourceChangeResponse) {
	if req.ProposedNewState.IsNull() {
		// destroy op
		resp.PlannedState = req.ProposedNewState

		// signal that this resource was properly planned for destruction,
		// verifying that the schema capabilities with PlanDestroy took effect.
		resp.PlannedPrivate = []byte("destroy planned")
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
		// make sure this was transferred from the plan action
		if string(req.PlannedPrivate) != "destroy planned" {
			resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("resource not planned for destroy, private data %q", req.PlannedPrivate))
		}

		resp.NewState = req.PlannedState
		resp.NewIdentity = req.PlannedIdentity
		return resp
	}

	m := req.PlannedState.AsValueMap()
	_, ok := m["id"]
	if !ok {
		m["id"] = cty.StringVal(time.Now().String())
	}
	resp.NewState = cty.ObjectVal(m)

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

func (p simple) ValidateEphemeralResourceConfig(req providers.ValidateEphemeralResourceConfigRequest) (resp providers.ValidateEphemeralResourceConfigResponse) {
	return resp
}

func (s simple) OpenEphemeralResource(req providers.OpenEphemeralResourceRequest) (resp providers.OpenEphemeralResourceResponse) {
	// we only have one type, so no need to check
	m := req.Config.AsValueMap()
	m["id"] = cty.StringVal("ephemeral secret")
	resp.Result = cty.ObjectVal(m)
	resp.Private = []byte("private data")
	resp.RenewAt = time.Now().Add(time.Second)
	return resp
}

func (s simple) RenewEphemeralResource(req providers.RenewEphemeralResourceRequest) (resp providers.RenewEphemeralResourceResponse) {
	log.Printf("[DEBUG] renewing ephemeral resource")
	if string(req.Private) != "private data" {
		resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("invalid private data %q, cannot renew ephemeral resource", req.Private))
	}
	resp.Private = req.Private
	resp.RenewAt = time.Now().Add(time.Second)
	return resp
}

func (s simple) CloseEphemeralResource(req providers.CloseEphemeralResourceRequest) (resp providers.CloseEphemeralResourceResponse) {
	log.Printf("[DEBUG] closing ephemeral resource")
	if string(req.Private) != "private data" {
		resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("invalid private data %q, cannot close ephemeral resource", req.Private))
	}
	return resp
}

func (s simple) CallFunction(req providers.CallFunctionRequest) (resp providers.CallFunctionResponse) {
	if req.FunctionName != "noop" {
		resp.Err = fmt.Errorf("CallFunction for undefined function %q", req.FunctionName)
		return resp
	}

	resp.Result = req.Arguments[0]
	return resp
}

func (s simple) ListResource(req providers.ListResourceRequest) (resp providers.ListResourceResponse) {
	vals := make([]cty.Value, 0)

	staticVal := cty.StringVal("static_value")
	m := req.Config.AsValueMap()
	if val, ok := m["value"]; ok && val != cty.NilVal {
		staticVal = val
	}

	obj := map[string]cty.Value{
		"display_name": cty.StringVal("static_display_name"),
		"identity": cty.ObjectVal(map[string]cty.Value{
			"id": cty.StringVal("static_id"),
		}),
	}
	if req.IncludeResourceObject {
		obj["state"] = cty.ObjectVal(map[string]cty.Value{
			"id":    cty.StringVal("static_id"),
			"value": staticVal,
		})
	}
	vals = append(vals, cty.ObjectVal(obj))

	resp.Result = cty.ObjectVal(map[string]cty.Value{
		"data":   cty.TupleVal(vals),
		"config": req.Config,
	})
	return
}

func (s simple) ValidateStateStoreConfig(req providers.ValidateStateStoreConfigRequest) providers.ValidateStateStoreConfigResponse {
	if req.TypeName == inMemStoreName {
		return s.inMem.ValidateStateStoreConfig(req)
	}
	if req.TypeName == fsStoreName {
		return s.fs.ValidateStateStoreConfig(req)
	}

	var resp providers.ValidateStateStoreConfigResponse
	resp.Diagnostics.Append(fmt.Errorf("unsupported state store type %q", req.TypeName))
	return resp
}

func (s simple) ConfigureStateStore(req providers.ConfigureStateStoreRequest) providers.ConfigureStateStoreResponse {
	if req.TypeName == inMemStoreName {
		return s.inMem.ConfigureStateStore(req)
	}
	if req.TypeName == fsStoreName {
		return s.fs.ConfigureStateStore(req)
	}

	var resp providers.ConfigureStateStoreResponse
	resp.Diagnostics.Append(fmt.Errorf("unsupported state store type %q", req.TypeName))
	return resp
}

func (s simple) ReadStateBytes(req providers.ReadStateBytesRequest) providers.ReadStateBytesResponse {
	if req.TypeName == inMemStoreName {
		return s.inMem.ReadStateBytes(req)
	}
	if req.TypeName == fsStoreName {
		return s.fs.ReadStateBytes(req)
	}

	var resp providers.ReadStateBytesResponse
	resp.Diagnostics.Append(fmt.Errorf("unsupported state store type %q", req.TypeName))
	return resp
}

func (s simple) WriteStateBytes(req providers.WriteStateBytesRequest) providers.WriteStateBytesResponse {
	if req.TypeName == inMemStoreName {
		return s.inMem.WriteStateBytes(req)
	}
	if req.TypeName == fsStoreName {
		return s.fs.WriteStateBytes(req)
	}

	var resp providers.WriteStateBytesResponse
	resp.Diagnostics.Append(fmt.Errorf("unsupported state store type %q", req.TypeName))
	return resp
}

func (s simple) LockState(req providers.LockStateRequest) providers.LockStateResponse {
	if req.TypeName == inMemStoreName {
		return s.inMem.LockState(req)
	}
	if req.TypeName == fsStoreName {
		return s.fs.LockState(req)
	}

	var resp providers.LockStateResponse
	resp.Diagnostics.Append(fmt.Errorf("unsupported state store type %q", req.TypeName))
	return resp
}

func (s simple) UnlockState(req providers.UnlockStateRequest) providers.UnlockStateResponse {
	if req.TypeName == inMemStoreName {
		return s.inMem.UnlockState(req)
	}
	if req.TypeName == fsStoreName {
		return s.fs.UnlockState(req)
	}

	var resp providers.UnlockStateResponse
	resp.Diagnostics.Append(fmt.Errorf("unsupported state store type %q", req.TypeName))
	return resp
}

func (s simple) GetStates(req providers.GetStatesRequest) providers.GetStatesResponse {
	if req.TypeName == inMemStoreName {
		return s.inMem.GetStates(req)
	}
	if req.TypeName == fsStoreName {
		return s.fs.GetStates(req)
	}

	var resp providers.GetStatesResponse
	resp.Diagnostics.Append(fmt.Errorf("unsupported state store type %q", req.TypeName))
	return resp
}

func (s simple) DeleteState(req providers.DeleteStateRequest) providers.DeleteStateResponse {
	if req.TypeName == inMemStoreName {
		return s.inMem.DeleteState(req)
	}
	if req.TypeName == fsStoreName {
		return s.fs.DeleteState(req)
	}

	var resp providers.DeleteStateResponse
	resp.Diagnostics.Append(fmt.Errorf("unsupported state store type %q", req.TypeName))
	return resp
}

func (s simple) PlanAction(providers.PlanActionRequest) providers.PlanActionResponse {
	// Our schema doesn't include any actions, so it should be
	// impossible to get here.
	panic("PlanAction on provider that didn't declare any actions")
}

func (s simple) InvokeAction(providers.InvokeActionRequest) providers.InvokeActionResponse {
	// Our schema doesn't include any actions, so it should be
	// impossible to get here.
	panic("InvokeAction on provider that didn't declare any actions")
}

func (s simple) ValidateActionConfig(providers.ValidateActionConfigRequest) providers.ValidateActionConfigResponse {
	// Our schema doesn't include any actions, so it should be
	// impossible to get here.
	panic("ValidateActionConfig on provider that didn't declare any actions")
}

func (s simple) Close() error {
	return nil
}

func (s simple) SetStateStoreChunkSize(typeName string, size int) {
	switch typeName {
	case inMemStoreName:
		s.inMem.SetStateStoreChunkSize(typeName, size)
	case fsStoreName:
		s.fs.SetStateStoreChunkSize(typeName, size)
	default:
		panic("SetStateStoreChunkSize called with unrecognized state store type name.")
	}
}
