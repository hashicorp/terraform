// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package grpcwrap

import (
	"context"
	"fmt"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	ctyjson "github.com/zclconf/go-cty/cty/json"
	"github.com/zclconf/go-cty/cty/msgpack"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hashicorp/terraform/internal/plugin/convert"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfplugin5"
)

// Provider wraps a providers.Interface to implement a grpc ProviderServer.
// This is useful for creating a test binary out of an internal provider
// implementation.
func Provider(p providers.Interface) tfplugin5.ProviderServer {
	return &provider{
		provider:        p,
		schema:          p.GetProviderSchema(),
		identitySchemas: p.GetResourceIdentitySchemas(),
	}
}

type provider struct {
	provider        providers.Interface
	schema          providers.GetProviderSchemaResponse
	identitySchemas providers.GetResourceIdentitySchemasResponse
}

func (p *provider) GetMetadata(_ context.Context, req *tfplugin5.GetMetadata_Request) (*tfplugin5.GetMetadata_Response, error) {
	return nil, status.Error(codes.Unimplemented, "GetMetadata is not implemented by core")
}

func (p *provider) GetSchema(_ context.Context, req *tfplugin5.GetProviderSchema_Request) (*tfplugin5.GetProviderSchema_Response, error) {
	resp := &tfplugin5.GetProviderSchema_Response{
		ResourceSchemas:          make(map[string]*tfplugin5.Schema),
		DataSourceSchemas:        make(map[string]*tfplugin5.Schema),
		EphemeralResourceSchemas: make(map[string]*tfplugin5.Schema),
		ListResourceSchemas:      make(map[string]*tfplugin5.Schema),
		ActionSchemas:            make(map[string]*tfplugin5.ActionSchema),
	}

	resp.Provider = &tfplugin5.Schema{
		Block: &tfplugin5.Schema_Block{},
	}
	if p.schema.Provider.Body != nil {
		resp.Provider.Block = convert.ConfigSchemaToProto(p.schema.Provider.Body)
	}

	resp.ProviderMeta = &tfplugin5.Schema{
		Block: &tfplugin5.Schema_Block{},
	}
	if p.schema.ProviderMeta.Body != nil {
		resp.ProviderMeta.Block = convert.ConfigSchemaToProto(p.schema.ProviderMeta.Body)
	}

	for typ, res := range p.schema.ResourceTypes {
		resp.ResourceSchemas[typ] = &tfplugin5.Schema{
			Version: res.Version,
			Block:   convert.ConfigSchemaToProto(res.Body),
		}
	}
	for typ, dat := range p.schema.DataSources {
		resp.DataSourceSchemas[typ] = &tfplugin5.Schema{
			Version: int64(dat.Version),
			Block:   convert.ConfigSchemaToProto(dat.Body),
		}
	}
	for typ, dat := range p.schema.EphemeralResourceTypes {
		resp.EphemeralResourceSchemas[typ] = &tfplugin5.Schema{
			Version: int64(dat.Version),
			Block:   convert.ConfigSchemaToProto(dat.Body),
		}
	}
	for typ, dat := range p.schema.ListResourceTypes {
		resp.ListResourceSchemas[typ] = &tfplugin5.Schema{
			Version: int64(dat.Version),
			Block:   convert.ConfigSchemaToProto(dat.Body),
		}
	}
	if decls, err := convert.FunctionDeclsToProto(p.schema.Functions); err == nil {
		resp.Functions = decls
	} else {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	for typ, act := range p.schema.Actions {
		newAct := tfplugin5.ActionSchema{
			Schema: &tfplugin5.Schema{
				Block: convert.ConfigSchemaToProto(act.ConfigSchema),
			},
		}

		if act.Unlinked != nil {
			newAct.Type = &tfplugin5.ActionSchema_Unlinked_{}
		} else if act.Lifecycle != nil {
			newAct.Type = &tfplugin5.ActionSchema_Lifecycle_{
				Lifecycle: &tfplugin5.ActionSchema_Lifecycle{
					Executes:       convert.ExecutionOrderToProto(act.Lifecycle.Exectues),
					LinkedResource: convert.LinkedResourceToProto(act.Lifecycle.LinkedResource),
				},
			}
		} else if act.Linked != nil {
			newAct.Type = &tfplugin5.ActionSchema_Linked_{
				Linked: &tfplugin5.ActionSchema_Linked{
					LinkedResources: convert.LinkedResourcesToProto(act.Linked.LinkedResources),
				},
			}
		}
		resp.ActionSchemas[typ] = &newAct
	}

	resp.ServerCapabilities = &tfplugin5.ServerCapabilities{
		GetProviderSchemaOptional: p.schema.ServerCapabilities.GetProviderSchemaOptional,
		PlanDestroy:               p.schema.ServerCapabilities.PlanDestroy,
		MoveResourceState:         p.schema.ServerCapabilities.MoveResourceState,
	}

	// include any diagnostics from the original GetSchema call
	resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, p.schema.Diagnostics)

	return resp, nil
}

func (p *provider) PrepareProviderConfig(_ context.Context, req *tfplugin5.PrepareProviderConfig_Request) (*tfplugin5.PrepareProviderConfig_Response, error) {
	resp := &tfplugin5.PrepareProviderConfig_Response{}
	ty := p.schema.Provider.Body.ImpliedType()

	configVal, err := decodeDynamicValue(req.Config, ty)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	prepareResp := p.provider.ValidateProviderConfig(providers.ValidateProviderConfigRequest{
		Config: configVal,
	})

	// the PreparedConfig value is no longer used
	resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, prepareResp.Diagnostics)
	return resp, nil
}

func (p *provider) ValidateResourceTypeConfig(_ context.Context, req *tfplugin5.ValidateResourceTypeConfig_Request) (*tfplugin5.ValidateResourceTypeConfig_Response, error) {
	resp := &tfplugin5.ValidateResourceTypeConfig_Response{}
	ty := p.schema.ResourceTypes[req.TypeName].Body.ImpliedType()

	configVal, err := decodeDynamicValue(req.Config, ty)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	validateResp := p.provider.ValidateResourceConfig(providers.ValidateResourceConfigRequest{
		TypeName: req.TypeName,
		Config:   configVal,
		ClientCapabilities: providers.ClientCapabilities{
			DeferralAllowed:            true,
			WriteOnlyAttributesAllowed: true,
		},
	})

	resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, validateResp.Diagnostics)
	return resp, nil
}

func (p *provider) ValidateDataSourceConfig(_ context.Context, req *tfplugin5.ValidateDataSourceConfig_Request) (*tfplugin5.ValidateDataSourceConfig_Response, error) {
	resp := &tfplugin5.ValidateDataSourceConfig_Response{}
	ty := p.schema.DataSources[req.TypeName].Body.ImpliedType()

	configVal, err := decodeDynamicValue(req.Config, ty)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	validateResp := p.provider.ValidateDataResourceConfig(providers.ValidateDataResourceConfigRequest{
		TypeName: req.TypeName,
		Config:   configVal,
	})

	resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, validateResp.Diagnostics)
	return resp, nil
}

func (p *provider) ValidateEphemeralResourceConfig(_ context.Context, req *tfplugin5.ValidateEphemeralResourceConfig_Request) (*tfplugin5.ValidateEphemeralResourceConfig_Response, error) {
	resp := &tfplugin5.ValidateEphemeralResourceConfig_Response{}
	ty := p.schema.DataSources[req.TypeName].Body.ImpliedType()

	configVal, err := decodeDynamicValue(req.Config, ty)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	validateResp := p.provider.ValidateEphemeralResourceConfig(providers.ValidateEphemeralResourceConfigRequest{
		TypeName: req.TypeName,
		Config:   configVal,
	})

	resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, validateResp.Diagnostics)
	return resp, nil
}

func (p *provider) ValidateListResourceConfig(_ context.Context, req *tfplugin5.ValidateListResourceConfig_Request) (*tfplugin5.ValidateListResourceConfig_Response, error) {
	resp := &tfplugin5.ValidateListResourceConfig_Response{}
	ty := p.schema.ListResourceTypes[req.TypeName].Body.ImpliedType()

	configVal, err := decodeDynamicValue(req.Config, ty)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	validateResp := p.provider.ValidateListResourceConfig(providers.ValidateListResourceConfigRequest{
		TypeName: req.TypeName,
		Config:   configVal,
	})

	resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, validateResp.Diagnostics)
	return resp, nil
}

func (p *provider) UpgradeResourceState(_ context.Context, req *tfplugin5.UpgradeResourceState_Request) (*tfplugin5.UpgradeResourceState_Response, error) {
	resp := &tfplugin5.UpgradeResourceState_Response{}
	ty := p.schema.ResourceTypes[req.TypeName].Body.ImpliedType()

	upgradeResp := p.provider.UpgradeResourceState(providers.UpgradeResourceStateRequest{
		TypeName:     req.TypeName,
		Version:      req.Version,
		RawStateJSON: req.RawState.Json,
	})

	resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, upgradeResp.Diagnostics)
	if upgradeResp.Diagnostics.HasErrors() {
		return resp, nil
	}

	dv, err := encodeDynamicValue(upgradeResp.UpgradedState, ty)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	resp.UpgradedState = dv

	return resp, nil
}

func (p *provider) Configure(_ context.Context, req *tfplugin5.Configure_Request) (*tfplugin5.Configure_Response, error) {
	resp := &tfplugin5.Configure_Response{}
	ty := p.schema.Provider.Body.ImpliedType()

	configVal, err := decodeDynamicValue(req.Config, ty)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	configureResp := p.provider.ConfigureProvider(providers.ConfigureProviderRequest{
		TerraformVersion: req.TerraformVersion,
		Config:           configVal,
	})

	resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, configureResp.Diagnostics)
	return resp, nil
}

func (p *provider) ReadResource(_ context.Context, req *tfplugin5.ReadResource_Request) (*tfplugin5.ReadResource_Response, error) {
	resp := &tfplugin5.ReadResource_Response{}
	resSchema := p.schema.ResourceTypes[req.TypeName]
	ty := resSchema.Body.ImpliedType()

	stateVal, err := decodeDynamicValue(req.CurrentState, ty)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	metaTy := p.schema.ProviderMeta.Body.ImpliedType()
	metaVal, err := decodeDynamicValue(req.ProviderMeta, metaTy)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	var currentIdentity cty.Value
	if req.CurrentIdentity != nil && req.CurrentIdentity.IdentityData != nil {
		if resSchema.Identity == nil {
			return resp, fmt.Errorf("identity schema not found for type %s", req.TypeName)
		}
		currentIdentity, err = decodeDynamicValue(req.CurrentIdentity.IdentityData, resSchema.Identity.ImpliedType())
		if err != nil {
			resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
			return resp, nil
		}
	}

	readResp := p.provider.ReadResource(providers.ReadResourceRequest{
		TypeName:        req.TypeName,
		PriorState:      stateVal,
		Private:         req.Private,
		ProviderMeta:    metaVal,
		CurrentIdentity: currentIdentity,
	})
	resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, readResp.Diagnostics)
	if readResp.Diagnostics.HasErrors() {
		return resp, nil
	}
	resp.Private = readResp.Private

	dv, err := encodeDynamicValue(readResp.NewState, ty)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}
	resp.NewState = dv

	if !readResp.Identity.IsNull() {
		if resSchema.Identity == nil {
			return resp, fmt.Errorf("identity schema not found for type %s", req.TypeName)
		}

		identity, err := encodeDynamicValue(readResp.Identity, resSchema.Identity.ImpliedType())
		if err != nil {
			resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
			return resp, nil
		}
		resp.NewIdentity = &tfplugin5.ResourceIdentityData{
			IdentityData: identity,
		}
	}
	return resp, nil
}

func (p *provider) PlanResourceChange(_ context.Context, req *tfplugin5.PlanResourceChange_Request) (*tfplugin5.PlanResourceChange_Response, error) {
	resp := &tfplugin5.PlanResourceChange_Response{}
	resSchema := p.schema.ResourceTypes[req.TypeName]
	ty := resSchema.Body.ImpliedType()

	priorStateVal, err := decodeDynamicValue(req.PriorState, ty)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	proposedStateVal, err := decodeDynamicValue(req.ProposedNewState, ty)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	configVal, err := decodeDynamicValue(req.Config, ty)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	metaTy := p.schema.ProviderMeta.Body.ImpliedType()
	metaVal, err := decodeDynamicValue(req.ProviderMeta, metaTy)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	var priorIdentity cty.Value
	if req.PriorIdentity != nil && req.PriorIdentity.IdentityData != nil {

		if resSchema.Identity == nil {
			return resp, fmt.Errorf("identity schema not found for type %s", req.TypeName)
		}

		priorIdentity, err = decodeDynamicValue(req.PriorIdentity.IdentityData, resSchema.Identity.ImpliedType())
		if err != nil {
			resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
			return resp, nil
		}
	}

	planResp := p.provider.PlanResourceChange(providers.PlanResourceChangeRequest{
		TypeName:         req.TypeName,
		PriorState:       priorStateVal,
		ProposedNewState: proposedStateVal,
		Config:           configVal,
		PriorPrivate:     req.PriorPrivate,
		ProviderMeta:     metaVal,
		PriorIdentity:    priorIdentity,
	})
	resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, planResp.Diagnostics)
	if planResp.Diagnostics.HasErrors() {
		return resp, nil
	}

	resp.PlannedPrivate = planResp.PlannedPrivate

	resp.PlannedState, err = encodeDynamicValue(planResp.PlannedState, ty)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	for _, path := range planResp.RequiresReplace {
		resp.RequiresReplace = append(resp.RequiresReplace, convert.PathToAttributePath(path))
	}

	if !planResp.PlannedIdentity.IsNull() {

		if resSchema.Identity == nil {
			return resp, fmt.Errorf("identity schema not found for type %s", req.TypeName)
		}

		plannedIdentity, err := encodeDynamicValue(planResp.PlannedIdentity, resSchema.Identity.ImpliedType())
		if err != nil {
			resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
			return resp, nil
		}

		resp.PlannedIdentity = &tfplugin5.ResourceIdentityData{
			IdentityData: plannedIdentity,
		}
	}

	return resp, nil
}

func (p *provider) ApplyResourceChange(_ context.Context, req *tfplugin5.ApplyResourceChange_Request) (*tfplugin5.ApplyResourceChange_Response, error) {
	resp := &tfplugin5.ApplyResourceChange_Response{}
	resSchema := p.schema.ResourceTypes[req.TypeName]
	ty := resSchema.Body.ImpliedType()

	priorStateVal, err := decodeDynamicValue(req.PriorState, ty)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	plannedStateVal, err := decodeDynamicValue(req.PlannedState, ty)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	configVal, err := decodeDynamicValue(req.Config, ty)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	metaTy := p.schema.ProviderMeta.Body.ImpliedType()
	metaVal, err := decodeDynamicValue(req.ProviderMeta, metaTy)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	var plannedIdentity cty.Value
	if req.PlannedIdentity != nil && req.PlannedIdentity.IdentityData != nil {
		if resSchema.Identity == nil {
			return resp, fmt.Errorf("identity schema not found for type %s", req.TypeName)
		}

		plannedIdentity, err = decodeDynamicValue(req.PlannedIdentity.IdentityData, resSchema.Identity.ImpliedType())
		if err != nil {
			resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
			return resp, nil
		}
	}

	applyResp := p.provider.ApplyResourceChange(providers.ApplyResourceChangeRequest{
		TypeName:        req.TypeName,
		PriorState:      priorStateVal,
		PlannedState:    plannedStateVal,
		Config:          configVal,
		PlannedPrivate:  req.PlannedPrivate,
		ProviderMeta:    metaVal,
		PlannedIdentity: plannedIdentity,
	})

	resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, applyResp.Diagnostics)
	if applyResp.Diagnostics.HasErrors() {
		return resp, nil
	}
	resp.Private = applyResp.Private
	resp.NewState, err = encodeDynamicValue(applyResp.NewState, ty)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	if !applyResp.NewIdentity.IsNull() {
		if resSchema.Identity == nil {
			return resp, fmt.Errorf("identity schema not found for type %s", req.TypeName)
		}

		newIdentity, err := encodeDynamicValue(applyResp.NewIdentity, resSchema.Identity.ImpliedType())
		if err != nil {
			resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
			return resp, nil
		}
		resp.NewIdentity = &tfplugin5.ResourceIdentityData{
			IdentityData: newIdentity,
		}
	}

	return resp, nil
}

func (p *provider) ImportResourceState(_ context.Context, req *tfplugin5.ImportResourceState_Request) (*tfplugin5.ImportResourceState_Response, error) {
	resp := &tfplugin5.ImportResourceState_Response{}
	var identity cty.Value
	var err error
	if req.Identity != nil && req.Identity.IdentityData != nil {
		resSchema := p.schema.ResourceTypes[req.TypeName]

		if resSchema.Identity == nil {
			return resp, fmt.Errorf("identity schema not found for type %s", req.TypeName)
		}

		identity, err = decodeDynamicValue(req.Identity.IdentityData, resSchema.Identity.ImpliedType())
		if err != nil {
			resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
			return resp, nil
		}
	}

	importResp := p.provider.ImportResourceState(providers.ImportResourceStateRequest{
		TypeName: req.TypeName,
		ID:       req.Id,
		Identity: identity,
	})
	resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, importResp.Diagnostics)

	for _, res := range importResp.ImportedResources {
		importSchema := p.schema.ResourceTypes[res.TypeName]
		ty := importSchema.Body.ImpliedType()
		state, err := encodeDynamicValue(res.State, ty)
		if err != nil {
			resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
			continue
		}

		resource := &tfplugin5.ImportResourceState_ImportedResource{
			TypeName: res.TypeName,
			State:    state,
			Private:  res.Private,
		}

		if !res.Identity.IsNull() {
			if importSchema.Identity == nil {
				return nil, fmt.Errorf("identity schema not found for type %s", res.TypeName)
			}
			identity, err := encodeDynamicValue(res.Identity, importSchema.Identity.ImpliedType())
			if err != nil {
				resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
				continue
			}
			resource.Identity = &tfplugin5.ResourceIdentityData{
				IdentityData: identity,
			}
		}

		resp.ImportedResources = append(resp.ImportedResources, resource)
	}

	return resp, nil
}

func (p *provider) MoveResourceState(_ context.Context, request *tfplugin5.MoveResourceState_Request) (*tfplugin5.MoveResourceState_Response, error) {
	resp := &tfplugin5.MoveResourceState_Response{}

	var sourceIdentity []byte
	var err error
	if request.SourceIdentity != nil && len(request.SourceIdentity.Json) > 0 {
		sourceIdentity = request.SourceIdentity.Json
	}

	moveResp := p.provider.MoveResourceState(providers.MoveResourceStateRequest{
		SourceProviderAddress: request.SourceProviderAddress,
		SourceTypeName:        request.SourceTypeName,
		SourceSchemaVersion:   request.SourceSchemaVersion,
		SourceStateJSON:       request.SourceState.Json,
		SourcePrivate:         request.SourcePrivate,
		SourceIdentity:        sourceIdentity,
		TargetTypeName:        request.TargetTypeName,
	})
	resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, moveResp.Diagnostics)
	if moveResp.Diagnostics.HasErrors() {
		return resp, nil
	}

	targetSchema := p.schema.ResourceTypes[request.TargetTypeName]
	targetType := targetSchema.Body.ImpliedType()
	targetState, err := encodeDynamicValue(moveResp.TargetState, targetType)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}
	resp.TargetState = targetState
	resp.TargetPrivate = moveResp.TargetPrivate

	if !moveResp.TargetIdentity.IsNull() {
		if targetSchema.Identity == nil {
			return resp, fmt.Errorf("identity schema not found for type %s", request.TargetTypeName)
		}
		targetIdentity, err := encodeDynamicValue(moveResp.TargetIdentity, targetSchema.Identity.ImpliedType())
		if err != nil {
			resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
			return resp, nil
		}
		resp.TargetIdentity = &tfplugin5.ResourceIdentityData{
			IdentityData: targetIdentity,
		}
	}

	return resp, nil
}

func (p *provider) ReadDataSource(_ context.Context, req *tfplugin5.ReadDataSource_Request) (*tfplugin5.ReadDataSource_Response, error) {
	resp := &tfplugin5.ReadDataSource_Response{}
	ty := p.schema.DataSources[req.TypeName].Body.ImpliedType()

	configVal, err := decodeDynamicValue(req.Config, ty)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	metaTy := p.schema.ProviderMeta.Body.ImpliedType()
	metaVal, err := decodeDynamicValue(req.ProviderMeta, metaTy)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	readResp := p.provider.ReadDataSource(providers.ReadDataSourceRequest{
		TypeName:     req.TypeName,
		Config:       configVal,
		ProviderMeta: metaVal,
	})
	resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, readResp.Diagnostics)
	if readResp.Diagnostics.HasErrors() {
		return resp, nil
	}

	resp.State, err = encodeDynamicValue(readResp.State, ty)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	return resp, nil
}

func (p *provider) OpenEphemeralResource(_ context.Context, req *tfplugin5.OpenEphemeralResource_Request) (*tfplugin5.OpenEphemeralResource_Response, error) {
	panic("unimplemented")
}

func (p *provider) RenewEphemeralResource(_ context.Context, req *tfplugin5.RenewEphemeralResource_Request) (*tfplugin5.RenewEphemeralResource_Response, error) {
	panic("unimplemented")
}

func (p *provider) CloseEphemeralResource(_ context.Context, req *tfplugin5.CloseEphemeralResource_Request) (*tfplugin5.CloseEphemeralResource_Response, error) {
	panic("unimplemented")
}

func (p *provider) GetFunctions(context.Context, *tfplugin5.GetFunctions_Request) (*tfplugin5.GetFunctions_Response, error) {
	panic("unimplemented")
}

func (p *provider) CallFunction(_ context.Context, req *tfplugin5.CallFunction_Request) (*tfplugin5.CallFunction_Response, error) {
	var err error
	resp := &tfplugin5.CallFunction_Response{}

	funcSchema := p.schema.Functions[req.Name]

	var args []cty.Value
	if len(req.Arguments) != 0 {
		args = make([]cty.Value, len(req.Arguments))
		for i, rawArg := range req.Arguments {
			idx := int64(i)

			var argTy cty.Type
			if i < len(funcSchema.Parameters) {
				argTy = funcSchema.Parameters[i].Type
			} else {
				if funcSchema.VariadicParameter == nil {

					resp.Error = &tfplugin5.FunctionError{
						Text:             "too many arguments for non-variadic function",
						FunctionArgument: &idx,
					}
					return resp, nil
				}
				argTy = funcSchema.VariadicParameter.Type
			}

			argVal, err := decodeDynamicValue(rawArg, argTy)
			if err != nil {
				resp.Error = &tfplugin5.FunctionError{
					Text:             err.Error(),
					FunctionArgument: &idx,
				}
				return resp, nil
			}

			args[i] = argVal
		}
	}

	callResp := p.provider.CallFunction(providers.CallFunctionRequest{
		FunctionName: req.Name,
		Arguments:    args,
	})

	if callResp.Err != nil {
		resp.Error = &tfplugin5.FunctionError{
			Text: callResp.Err.Error(),
		}

		if argErr, ok := callResp.Err.(function.ArgError); ok {
			idx := int64(argErr.Index)
			resp.Error.FunctionArgument = &idx
		}

		return resp, nil
	}

	resp.Result, err = encodeDynamicValue(callResp.Result, funcSchema.ReturnType)
	if err != nil {
		resp.Error = &tfplugin5.FunctionError{
			Text: err.Error(),
		}

		return resp, nil
	}

	return resp, nil
}

func (p *provider) GetResourceIdentitySchemas(_ context.Context, req *tfplugin5.GetResourceIdentitySchemas_Request) (*tfplugin5.GetResourceIdentitySchemas_Response, error) {
	resp := &tfplugin5.GetResourceIdentitySchemas_Response{
		IdentitySchemas: map[string]*tfplugin5.ResourceIdentitySchema{},
		Diagnostics:     []*tfplugin5.Diagnostic{},
	}

	for name, schema := range p.identitySchemas.IdentityTypes {
		resp.IdentitySchemas[name] = convert.ResourceIdentitySchemaToProto(schema)
	}

	resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, p.identitySchemas.Diagnostics)
	return resp, nil
}

func (p *provider) UpgradeResourceIdentity(_ context.Context, req *tfplugin5.UpgradeResourceIdentity_Request) (*tfplugin5.UpgradeResourceIdentity_Response, error) {
	resp := &tfplugin5.UpgradeResourceIdentity_Response{}
	resource, ok := p.schema.ResourceTypes[req.TypeName]
	if !ok {
		return nil, fmt.Errorf("resource identity schema not found for type %q", req.TypeName)
	}
	ty := resource.Identity.ImpliedType()
	upgradeResp := p.provider.UpgradeResourceIdentity(providers.UpgradeResourceIdentityRequest{
		TypeName:        req.TypeName,
		Version:         req.Version,
		RawIdentityJSON: req.RawIdentity.Json,
	})
	resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, upgradeResp.Diagnostics)
	if upgradeResp.Diagnostics.HasErrors() {
		return resp, nil
	}

	dv, err := encodeDynamicValue(upgradeResp.UpgradedIdentity, ty)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}
	resp.UpgradedIdentity = &tfplugin5.ResourceIdentityData{
		IdentityData: dv,
	}
	return resp, nil
}

func (p *provider) ListResource(req *tfplugin5.ListResource_Request, res tfplugin5.Provider_ListResourceServer) error {
	resourceSchema, ok := p.schema.ResourceTypes[req.TypeName]
	if !ok {
		return fmt.Errorf("resource schema not found for type %q", req.TypeName)
	}
	listSchema, ok := p.schema.ListResourceTypes[req.TypeName]
	if !ok {
		return fmt.Errorf("list resource schema not found for type %q", req.TypeName)
	}
	cfg, err := decodeDynamicValue(req.Config, listSchema.Body.ImpliedType())
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "failed to decode config: %v", err)
	}
	resp := p.provider.ListResource(providers.ListResourceRequest{
		TypeName:              req.TypeName,
		Config:                cfg,
		Limit:                 req.Limit,
		IncludeResourceObject: req.IncludeResourceObject,
	})
	if resp.Diagnostics.HasErrors() {
		return resp.Diagnostics.Err()
	}
	if !resp.Result.Type().HasAttribute("data") {
		return status.Errorf(codes.Internal, "list resource response missing 'data' attribute")
	}
	data := resp.Result.GetAttr("data")
	if data.IsNull() || !data.CanIterateElements() {
		return status.Errorf(codes.Internal, "list resource response 'data' attribute is invalid or null")
	}

	for iter := data.ElementIterator(); iter.Next(); {
		_, item := iter.Element()
		var stateVal *tfplugin5.DynamicValue
		if item.Type().HasAttribute("state") {
			state := item.GetAttr("state")
			var err error
			stateVal, err = encodeDynamicValue(state, resourceSchema.Body.ImpliedType())
			if err != nil {
				return status.Errorf(codes.Internal, "failed to encode list resource item state: %v", err)
			}
		}
		if !item.Type().HasAttribute("identity") {
			return status.Errorf(codes.Internal, "list resource item missing 'identity' attribute")
		}
		identity := item.GetAttr("identity")
		var identityVal *tfplugin5.DynamicValue
		identityVal, err = encodeDynamicValue(identity, resourceSchema.Identity.ImpliedType())
		if err != nil {
			return status.Errorf(codes.Internal, "failed to encode list resource item identity: %v", err)
		}

		var displayName string
		if item.Type().HasAttribute("display_name") {
			displayName = item.GetAttr("display_name").AsString()
		}

		res.Send(&tfplugin5.ListResource_Event{
			Identity:       &tfplugin5.ResourceIdentityData{IdentityData: identityVal},
			ResourceObject: stateVal,
			DisplayName:    displayName,
		})
	}

	return nil
}

func (p *provider) PlanAction(_ context.Context, req *tfplugin5.PlanAction_Request) (*tfplugin5.PlanAction_Response, error) {
	resp := &tfplugin5.PlanAction_Response{}

	actionSchema, ok := p.schema.Actions[req.ActionType]
	if !ok {
		return nil, fmt.Errorf("action schema not found for action %q", req.ActionType)
	}

	ty := actionSchema.ConfigSchema.ImpliedType()
	configVal, err := decodeDynamicValue(req.Config, ty)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	linkedResouceSchemas := actionSchema.LinkedResources()
	inputLinkedResources := make([]providers.LinkedResourcePlanData, 0, len(req.LinkedResources))

	for i, lr := range req.LinkedResources {
		resourceSchema, ok := p.schema.ResourceTypes[linkedResouceSchemas[i].TypeName]
		if !ok {
			return nil, fmt.Errorf("linked resource schema not found for type %q in action %q", linkedResouceSchemas[i].TypeName, req.ActionType)
		}

		linkedResourceTy := resourceSchema.Body.ImpliedType()
		linkedResourceIdentityTy := resourceSchema.Identity.ImpliedType()

		priorState, err := decodeDynamicValue(lr.PriorState, linkedResourceTy)
		if err != nil {
			return nil, fmt.Errorf("failed to decode prior state for linked resource #%d (%q) in action %q: %w", i, linkedResouceSchemas[i].TypeName, req.ActionType, err)
		}

		config, err := decodeDynamicValue(lr.Config, linkedResourceTy)
		if err != nil {
			return nil, fmt.Errorf("failed to decode config for linked resource #%d (%q) in action %q: %w", i, linkedResouceSchemas[i].TypeName, req.ActionType, err)
		}

		var priorIdentity cty.Value
		if lr.PriorIdentity != nil && lr.PriorIdentity.IdentityData != nil {
			priorIdentity, err = decodeDynamicValue(lr.PriorIdentity.IdentityData, linkedResourceIdentityTy)
			if err != nil {
				return nil, fmt.Errorf("failed to decode prior identity for linked resource #%d (%q) in action %q: %w", i, linkedResouceSchemas[i].TypeName, req.ActionType, err)
			}
		} else {
			priorIdentity = cty.NullVal(linkedResourceIdentityTy)
		}

		plannedState, err := decodeDynamicValue(lr.PlannedState, linkedResourceTy)
		if err != nil {
			return nil, fmt.Errorf("failed to decode planned state for linked resource #%d (%q) in action %q: %w", i, linkedResouceSchemas[i].TypeName, req.ActionType, err)
		}

		inputLinkedResources = append(inputLinkedResources, providers.LinkedResourcePlanData{
			PriorState:    priorState,
			Config:        config,
			PriorIdentity: priorIdentity,
			PlannedState:  plannedState,
		})
	}

	planResp := p.provider.PlanAction(providers.PlanActionRequest{
		ActionType:         req.ActionType,
		ProposedActionData: configVal,
		LinkedResources:    inputLinkedResources,
	})

	resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, planResp.Diagnostics)
	if planResp.Diagnostics.HasErrors() {
		return resp, nil
	}

	resp.Deferred = convert.DeferredToProto(planResp.Deferred)

	linkedResources := make([]*tfplugin5.PlanAction_Response_LinkedResource, 0, len(planResp.LinkedResources))
	for _, linked := range planResp.LinkedResources {
		plannedState, err := encodeDynamicValue(linked.PlannedState, linked.PlannedState.Type())
		if err != nil {
			resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
			continue
		}

		plannedIdentity, err := encodeDynamicValue(linked.PlannedIdentity, linked.PlannedIdentity.Type())
		if err != nil {
			resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
			continue
		}
		linkedResources = append(linkedResources, &tfplugin5.PlanAction_Response_LinkedResource{
			PlannedState: plannedState,
			PlannedIdentity: &tfplugin5.ResourceIdentityData{
				IdentityData: plannedIdentity,
			},
		})
	}
	resp.LinkedResources = linkedResources

	return resp, nil
}

func (p *provider) InvokeAction(req *tfplugin5.InvokeAction_Request, server tfplugin5.Provider_InvokeActionServer) error {

	actionSchema, ok := p.schema.Actions[req.ActionType]
	if !ok {
		return fmt.Errorf("action schema not found for action %q", req.ActionType)
	}

	ty := actionSchema.ConfigSchema.ImpliedType()
	configVal, err := decodeDynamicValue(req.Config, ty)
	if err != nil {
		return err
	}

	linkedResourceData := make([]providers.LinkedResourceInvokeData, 0, len(req.LinkedResources))
	linkedResourceSchemas := actionSchema.LinkedResources()
	if len(linkedResourceSchemas) != len(req.LinkedResources) {
		return fmt.Errorf("expected %d linked resources for action %q, got %d", len(linkedResourceSchemas), req.ActionType, len(req.LinkedResources))
	}
	for i, lr := range req.LinkedResources {
		resourceSchema, ok := p.schema.ResourceTypes[linkedResourceSchemas[i].TypeName]
		if !ok {
			return fmt.Errorf("linked resource schema not found for type %q in action %q", linkedResourceSchemas[i].TypeName, req.ActionType)
		}

		linkedResourceTy := resourceSchema.Body.ImpliedType()
		linkedResourceIdentityTy := resourceSchema.Identity.ImpliedType()

		priorState, err := decodeDynamicValue(lr.PriorState, linkedResourceTy)
		if err != nil {
			return fmt.Errorf("failed to decode prior state for linked resource #%d (%q) in action %q: %w", i, linkedResourceSchemas[i].TypeName, req.ActionType, err)
		}

		plannedState, err := decodeDynamicValue(lr.PlannedState, linkedResourceTy)
		if err != nil {
			return fmt.Errorf("failed to decode planned state for linked resource #%d (%q) in action %q: %w", i, linkedResourceSchemas[i].TypeName, req.ActionType, err)
		}

		config, err := decodeDynamicValue(lr.Config, linkedResourceTy)
		if err != nil {
			return fmt.Errorf("failed to decode config for linked resource #%d (%q) in action %q: %w", i, linkedResourceSchemas[i].TypeName, req.ActionType, err)
		}

		plannedIdentity := cty.NullVal(linkedResourceIdentityTy)
		if lr.PlannedIdentity != nil && lr.PlannedIdentity.IdentityData != nil {
			plannedIdentity, err = decodeDynamicValue(lr.PlannedIdentity.IdentityData, linkedResourceIdentityTy)
			if err != nil {
				return fmt.Errorf("failed to decode planned identity for linked resource #%d (%q) in action %q: %w", i, linkedResourceSchemas[i].TypeName, req.ActionType, err)
			}
		}

		linkedResourceData = append(linkedResourceData, providers.LinkedResourceInvokeData{
			PriorState:      priorState,
			PlannedState:    plannedState,
			Config:          config,
			PlannedIdentity: plannedIdentity,
		})
	}

	invokeResp := p.provider.InvokeAction(providers.InvokeActionRequest{
		ActionType:        req.ActionType,
		PlannedActionData: configVal,
		LinkedResources:   linkedResourceData,
	})

	if invokeResp.Diagnostics.HasErrors() {
		return invokeResp.Diagnostics.Err()
	}

	for invokeEvent := range invokeResp.Events {
		switch invokeEvt := invokeEvent.(type) {
		case providers.InvokeActionEvent_Progress:
			server.Send(&tfplugin5.InvokeAction_Event{
				Type: &tfplugin5.InvokeAction_Event_Progress_{
					Progress: &tfplugin5.InvokeAction_Event_Progress{
						Message: invokeEvt.Message,
					},
				},
			})

		case providers.InvokeActionEvent_Completed:
			completed := &tfplugin5.InvokeAction_Event_Completed{
				LinkedResources: []*tfplugin5.InvokeAction_Event_Completed_LinkedResource{},
			}
			completed.Diagnostics = convert.AppendProtoDiag(completed.Diagnostics, invokeEvt.Diagnostics)

			for _, lr := range invokeEvt.LinkedResources {
				newState, err := encodeDynamicValue(lr.NewState, lr.NewState.Type())
				if err != nil {
					return fmt.Errorf("failed to encode new state for linked resource: %w", err)
				}

				newIdentity, err := encodeDynamicValue(lr.NewIdentity, lr.NewIdentity.Type())
				if err != nil {
					return fmt.Errorf("failed to encode new identity for linked resource: %w", err)
				}

				completed.LinkedResources = append(completed.LinkedResources, &tfplugin5.InvokeAction_Event_Completed_LinkedResource{
					NewState: newState,
					NewIdentity: &tfplugin5.ResourceIdentityData{
						IdentityData: newIdentity,
					},
					RequiresReplace: lr.RequiresReplace,
				})
			}

			err := server.Send(&tfplugin5.InvokeAction_Event{
				Type: &tfplugin5.InvokeAction_Event_Completed_{
					Completed: completed,
				},
			})
			if err != nil {
				return fmt.Errorf("failed to send completed event: %w", err)
			}
		}

	}

	return nil
}

func (p *provider) Stop(context.Context, *tfplugin5.Stop_Request) (*tfplugin5.Stop_Response, error) {
	resp := &tfplugin5.Stop_Response{}
	err := p.provider.Stop()
	if err != nil {
		resp.Error = err.Error()
	}
	return resp, nil
}

// decode a DynamicValue from either the JSON or MsgPack encoding.
func decodeDynamicValue(v *tfplugin5.DynamicValue, ty cty.Type) (cty.Value, error) {
	// always return a valid value
	var err error
	res := cty.NullVal(ty)
	if v == nil {
		return res, nil
	}

	switch {
	case len(v.Msgpack) > 0:
		res, err = msgpack.Unmarshal(v.Msgpack, ty)
	case len(v.Json) > 0:
		res, err = ctyjson.Unmarshal(v.Json, ty)
	}
	return res, err
}

// encode a cty.Value into a DynamicValue msgpack payload.
func encodeDynamicValue(v cty.Value, ty cty.Type) (*tfplugin5.DynamicValue, error) {
	mp, err := msgpack.Marshal(v, ty)
	return &tfplugin5.DynamicValue{
		Msgpack: mp,
	}, err
}
