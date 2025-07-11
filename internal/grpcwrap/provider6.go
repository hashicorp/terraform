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
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/hashicorp/terraform/internal/plugin6/convert"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfplugin6"
)

// Provider6 wraps a providers.Interface to implement a grpc ProviderServer using
// plugin protocol v6. This is useful for creating a test binary out of an
// internal provider implementation.
func Provider6(p providers.Interface) tfplugin6.ProviderServer {
	return &provider6{
		provider:        p,
		schema:          p.GetProviderSchema(),
		identitySchemas: p.GetResourceIdentitySchemas(),
	}
}

type provider6 struct {
	provider        providers.Interface
	schema          providers.GetProviderSchemaResponse
	identitySchemas providers.GetResourceIdentitySchemasResponse
}

func (p *provider6) GetMetadata(_ context.Context, req *tfplugin6.GetMetadata_Request) (*tfplugin6.GetMetadata_Response, error) {
	return nil, status.Error(codes.Unimplemented, "GetMetadata is not implemented by core")
}

func (p *provider6) GetProviderSchema(_ context.Context, req *tfplugin6.GetProviderSchema_Request) (*tfplugin6.GetProviderSchema_Response, error) {
	resp := &tfplugin6.GetProviderSchema_Response{
		ResourceSchemas:          make(map[string]*tfplugin6.Schema),
		DataSourceSchemas:        make(map[string]*tfplugin6.Schema),
		EphemeralResourceSchemas: make(map[string]*tfplugin6.Schema),
		Functions:                make(map[string]*tfplugin6.Function),
		ListResourceSchemas:      make(map[string]*tfplugin6.Schema),
		StateStoreSchemas:        make(map[string]*tfplugin6.Schema),
		ActionSchemas:            make(map[string]*tfplugin6.ActionSchema),
	}

	resp.Provider = &tfplugin6.Schema{
		Block: &tfplugin6.Schema_Block{},
	}
	if p.schema.Provider.Body != nil {
		resp.Provider.Block = convert.ConfigSchemaToProto(p.schema.Provider.Body)
	}

	resp.ProviderMeta = &tfplugin6.Schema{
		Block: &tfplugin6.Schema_Block{},
	}
	if p.schema.ProviderMeta.Body != nil {
		resp.ProviderMeta.Block = convert.ConfigSchemaToProto(p.schema.ProviderMeta.Body)
	}

	for typ, res := range p.schema.ResourceTypes {
		resp.ResourceSchemas[typ] = &tfplugin6.Schema{
			Version: res.Version,
			Block:   convert.ConfigSchemaToProto(res.Body),
		}
	}
	for typ, dat := range p.schema.DataSources {
		resp.DataSourceSchemas[typ] = &tfplugin6.Schema{
			Version: int64(dat.Version),
			Block:   convert.ConfigSchemaToProto(dat.Body),
		}
	}
	for typ, dat := range p.schema.EphemeralResourceTypes {
		resp.EphemeralResourceSchemas[typ] = &tfplugin6.Schema{
			Version: int64(dat.Version),
			Block:   convert.ConfigSchemaToProto(dat.Body),
		}
	}
	for typ, dat := range p.schema.ListResourceTypes {
		resp.ListResourceSchemas[typ] = &tfplugin6.Schema{
			Version: int64(dat.Version),
			Block:   convert.ConfigSchemaToProto(dat.Body),
		}
	}
	for typ, dat := range p.schema.StateStores {
		resp.StateStoreSchemas[typ] = &tfplugin6.Schema{
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
		newAct := tfplugin6.ActionSchema{
			Schema: &tfplugin6.Schema{
				Block: convert.ConfigSchemaToProto(act.ConfigSchema),
			},
		}

		if act.Unlinked != nil {
			newAct.Type = &tfplugin6.ActionSchema_Unlinked_{}
		} else if act.Lifecycle != nil {
			newAct.Type = &tfplugin6.ActionSchema_Lifecycle_{
				Lifecycle: &tfplugin6.ActionSchema_Lifecycle{
					Executes:       convert.ExecutionOrderToProto(act.Lifecycle.Exectues),
					LinkedResource: convert.LinkedResourceToProto(act.Lifecycle.LinkedResource),
				},
			}
		} else if act.Linked != nil {
			newAct.Type = &tfplugin6.ActionSchema_Linked_{
				Linked: &tfplugin6.ActionSchema_Linked{
					LinkedResources: convert.LinkedResourcesToProto(act.Linked.LinkedResources),
				},
			}
		}
		resp.ActionSchemas[typ] = &newAct
	}

	resp.ServerCapabilities = &tfplugin6.ServerCapabilities{
		GetProviderSchemaOptional: p.schema.ServerCapabilities.GetProviderSchemaOptional,
		PlanDestroy:               p.schema.ServerCapabilities.PlanDestroy,
		MoveResourceState:         p.schema.ServerCapabilities.MoveResourceState,
	}

	// include any diagnostics from the original GetSchema call
	resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, p.schema.Diagnostics)

	return resp, nil
}

func (p *provider6) ValidateProviderConfig(_ context.Context, req *tfplugin6.ValidateProviderConfig_Request) (*tfplugin6.ValidateProviderConfig_Response, error) {
	resp := &tfplugin6.ValidateProviderConfig_Response{}
	ty := p.schema.Provider.Body.ImpliedType()

	configVal, err := decodeDynamicValue6(req.Config, ty)
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

func (p *provider6) ValidateResourceConfig(_ context.Context, req *tfplugin6.ValidateResourceConfig_Request) (*tfplugin6.ValidateResourceConfig_Response, error) {
	resp := &tfplugin6.ValidateResourceConfig_Response{}
	ty := p.schema.ResourceTypes[req.TypeName].Body.ImpliedType()

	configVal, err := decodeDynamicValue6(req.Config, ty)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	validateResp := p.provider.ValidateResourceConfig(providers.ValidateResourceConfigRequest{
		TypeName: req.TypeName,
		Config:   configVal,
	})

	resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, validateResp.Diagnostics)
	return resp, nil
}

func (p *provider6) ValidateDataResourceConfig(_ context.Context, req *tfplugin6.ValidateDataResourceConfig_Request) (*tfplugin6.ValidateDataResourceConfig_Response, error) {
	resp := &tfplugin6.ValidateDataResourceConfig_Response{}
	ty := p.schema.DataSources[req.TypeName].Body.ImpliedType()

	configVal, err := decodeDynamicValue6(req.Config, ty)
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

func (p *provider6) ValidateEphemeralResourceConfig(_ context.Context, req *tfplugin6.ValidateEphemeralResourceConfig_Request) (*tfplugin6.ValidateEphemeralResourceConfig_Response, error) {
	resp := &tfplugin6.ValidateEphemeralResourceConfig_Response{}
	ty := p.schema.EphemeralResourceTypes[req.TypeName].Body.ImpliedType()

	configVal, err := decodeDynamicValue6(req.Config, ty)
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

func (p *provider6) ValidateListResourceConfig(_ context.Context, req *tfplugin6.ValidateListResourceConfig_Request) (*tfplugin6.ValidateListResourceConfig_Response, error) {
	resp := &tfplugin6.ValidateListResourceConfig_Response{}
	ty := p.schema.ListResourceTypes[req.TypeName].Body.ImpliedType()

	configVal, err := decodeDynamicValue6(req.Config, ty)
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

func (p *provider6) UpgradeResourceState(_ context.Context, req *tfplugin6.UpgradeResourceState_Request) (*tfplugin6.UpgradeResourceState_Response, error) {
	resp := &tfplugin6.UpgradeResourceState_Response{}
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

	dv, err := encodeDynamicValue6(upgradeResp.UpgradedState, ty)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	resp.UpgradedState = dv

	return resp, nil
}

func (p *provider6) ConfigureProvider(_ context.Context, req *tfplugin6.ConfigureProvider_Request) (*tfplugin6.ConfigureProvider_Response, error) {
	resp := &tfplugin6.ConfigureProvider_Response{}
	ty := p.schema.Provider.Body.ImpliedType()

	configVal, err := decodeDynamicValue6(req.Config, ty)
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

func (p *provider6) ReadResource(_ context.Context, req *tfplugin6.ReadResource_Request) (*tfplugin6.ReadResource_Response, error) {
	resp := &tfplugin6.ReadResource_Response{}
	resSchema := p.schema.ResourceTypes[req.TypeName]
	ty := resSchema.Body.ImpliedType()

	stateVal, err := decodeDynamicValue6(req.CurrentState, ty)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	metaTy := p.schema.ProviderMeta.Body.ImpliedType()
	metaVal, err := decodeDynamicValue6(req.ProviderMeta, metaTy)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	var currentIdentity cty.Value
	if req.CurrentIdentity != nil && req.CurrentIdentity.IdentityData != nil {
		if resSchema.Identity == nil {
			return resp, fmt.Errorf("identity schema not found for type %s", req.TypeName)
		}

		currentIdentity, err = decodeDynamicValue6(req.CurrentIdentity.IdentityData, resSchema.Identity.ImpliedType())
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

	dv, err := encodeDynamicValue6(readResp.NewState, ty)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}
	resp.NewState = dv

	if !readResp.Identity.IsNull() {
		if resSchema.Identity == nil {
			return resp, fmt.Errorf("identity schema not found for type %s", req.TypeName)
		}
		newIdentity, err := encodeDynamicValue6(readResp.Identity, resSchema.Identity.ImpliedType())
		if err != nil {
			resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
			return resp, nil
		}
		resp.NewIdentity = &tfplugin6.ResourceIdentityData{
			IdentityData: newIdentity,
		}
	}

	return resp, nil
}

func (p *provider6) PlanResourceChange(_ context.Context, req *tfplugin6.PlanResourceChange_Request) (*tfplugin6.PlanResourceChange_Response, error) {
	resp := &tfplugin6.PlanResourceChange_Response{}
	resSchema := p.schema.ResourceTypes[req.TypeName]
	ty := resSchema.Body.ImpliedType()

	priorStateVal, err := decodeDynamicValue6(req.PriorState, ty)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	proposedStateVal, err := decodeDynamicValue6(req.ProposedNewState, ty)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	configVal, err := decodeDynamicValue6(req.Config, ty)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	metaTy := p.schema.ProviderMeta.Body.ImpliedType()
	metaVal, err := decodeDynamicValue6(req.ProviderMeta, metaTy)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	var priorIdentity cty.Value
	if req.PriorIdentity != nil && req.PriorIdentity.IdentityData != nil {
		if resSchema.Identity == nil {
			return resp, fmt.Errorf("identity schema not found for type %s", req.TypeName)
		}

		priorIdentity, err = decodeDynamicValue6(req.PriorIdentity.IdentityData, resSchema.Identity.ImpliedType())
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

	resp.PlannedState, err = encodeDynamicValue6(planResp.PlannedState, ty)
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

		plannedIdentityVal, err := encodeDynamicValue6(planResp.PlannedIdentity, resSchema.Identity.ImpliedType())
		if err != nil {
			resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
			return resp, nil
		}

		resp.PlannedIdentity = &tfplugin6.ResourceIdentityData{
			IdentityData: plannedIdentityVal,
		}
	}

	return resp, nil
}

func (p *provider6) ApplyResourceChange(_ context.Context, req *tfplugin6.ApplyResourceChange_Request) (*tfplugin6.ApplyResourceChange_Response, error) {
	resp := &tfplugin6.ApplyResourceChange_Response{}
	resSchema := p.schema.ResourceTypes[req.TypeName]
	ty := resSchema.Body.ImpliedType()

	priorStateVal, err := decodeDynamicValue6(req.PriorState, ty)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	plannedStateVal, err := decodeDynamicValue6(req.PlannedState, ty)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	configVal, err := decodeDynamicValue6(req.Config, ty)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	metaTy := p.schema.ProviderMeta.Body.ImpliedType()
	metaVal, err := decodeDynamicValue6(req.ProviderMeta, metaTy)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	var plannedIdentity cty.Value
	if req.PlannedIdentity != nil && req.PlannedIdentity.IdentityData != nil {
		if resSchema.Identity == nil {
			return resp, fmt.Errorf("identity schema not found for type %s", req.TypeName)
		}

		plannedIdentity, err = decodeDynamicValue6(req.PlannedIdentity.IdentityData, resSchema.Identity.ImpliedType())
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

	resp.NewState, err = encodeDynamicValue6(applyResp.NewState, ty)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	if !applyResp.NewIdentity.IsNull() {
		if resSchema.Identity == nil {
			return resp, fmt.Errorf("identity schema not found for type %s", req.TypeName)
		}
		newIdentity, err := encodeDynamicValue6(applyResp.NewIdentity, resSchema.Identity.ImpliedType())
		if err != nil {
			resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
			return resp, nil
		}
		resp.NewIdentity = &tfplugin6.ResourceIdentityData{
			IdentityData: newIdentity,
		}
	}

	return resp, nil
}

func (p *provider6) ImportResourceState(_ context.Context, req *tfplugin6.ImportResourceState_Request) (*tfplugin6.ImportResourceState_Response, error) {
	resp := &tfplugin6.ImportResourceState_Response{}

	resSchema := p.schema.ResourceTypes[req.TypeName]

	var identity cty.Value
	var err error
	if req.Identity != nil && req.Identity.IdentityData != nil {
		if resSchema.Identity == nil {
			return resp, fmt.Errorf("identity schema not found for type %s", req.TypeName)
		}
		identity, err = decodeDynamicValue6(req.Identity.IdentityData, resSchema.Identity.ImpliedType())
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
		state, err := encodeDynamicValue6(res.State, ty)
		if err != nil {
			resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
			continue
		}
		importedResource := &tfplugin6.ImportResourceState_ImportedResource{
			TypeName: res.TypeName,
			State:    state,
			Private:  res.Private,
		}
		if !res.Identity.IsNull() {
			if importSchema.Identity == nil {
				return nil, fmt.Errorf("identity schema not found for type %s", res.TypeName)
			}

			identity, err := encodeDynamicValue6(res.Identity, importSchema.Identity.ImpliedType())
			if err != nil {
				resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
				continue
			}

			importedResource.Identity = &tfplugin6.ResourceIdentityData{
				IdentityData: identity,
			}
		}

		resp.ImportedResources = append(resp.ImportedResources, importedResource)
	}

	return resp, nil
}

func (p *provider6) MoveResourceState(_ context.Context, request *tfplugin6.MoveResourceState_Request) (*tfplugin6.MoveResourceState_Response, error) {
	resp := &tfplugin6.MoveResourceState_Response{}

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
		TargetTypeName:        request.TargetTypeName,
		SourceIdentity:        sourceIdentity,
	})
	resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, moveResp.Diagnostics)
	if moveResp.Diagnostics.HasErrors() {
		return resp, nil
	}

	targetSchema := p.schema.ResourceTypes[request.TargetTypeName]
	targetType := targetSchema.Body.ImpliedType()
	targetState, err := encodeDynamicValue6(moveResp.TargetState, targetType)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	if !moveResp.TargetIdentity.IsNull() {
		if targetSchema.Identity == nil {
			return resp, fmt.Errorf("identity schema not found for type %s", request.TargetTypeName)
		}

		targetIdentity, err := encodeDynamicValue6(moveResp.TargetIdentity, targetSchema.Identity.ImpliedType())
		if err != nil {
			resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
			return resp, nil
		}

		resp.TargetIdentity = &tfplugin6.ResourceIdentityData{
			IdentityData: targetIdentity,
		}
	}

	resp.TargetState = targetState
	resp.TargetPrivate = moveResp.TargetPrivate
	return resp, nil
}

func (p *provider6) ReadDataSource(_ context.Context, req *tfplugin6.ReadDataSource_Request) (*tfplugin6.ReadDataSource_Response, error) {
	resp := &tfplugin6.ReadDataSource_Response{}
	ty := p.schema.DataSources[req.TypeName].Body.ImpliedType()

	configVal, err := decodeDynamicValue6(req.Config, ty)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	metaTy := p.schema.ProviderMeta.Body.ImpliedType()
	metaVal, err := decodeDynamicValue6(req.ProviderMeta, metaTy)
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

	resp.State, err = encodeDynamicValue6(readResp.State, ty)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	return resp, nil
}

func (p *provider6) OpenEphemeralResource(_ context.Context, req *tfplugin6.OpenEphemeralResource_Request) (*tfplugin6.OpenEphemeralResource_Response, error) {
	resp := &tfplugin6.OpenEphemeralResource_Response{}
	ty := p.schema.EphemeralResourceTypes[req.TypeName].Body.ImpliedType()

	configVal, err := decodeDynamicValue6(req.Config, ty)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	openResp := p.provider.OpenEphemeralResource(providers.OpenEphemeralResourceRequest{
		TypeName: req.TypeName,
		Config:   configVal,
	})
	resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, openResp.Diagnostics)
	if openResp.Diagnostics.HasErrors() {
		return resp, nil
	}

	resp.Result, err = encodeDynamicValue6(openResp.Result, ty)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}
	resp.Private = openResp.Private
	resp.RenewAt = timestamppb.New(openResp.RenewAt)

	return resp, nil
}

func (p *provider6) RenewEphemeralResource(_ context.Context, req *tfplugin6.RenewEphemeralResource_Request) (*tfplugin6.RenewEphemeralResource_Response, error) {
	resp := &tfplugin6.RenewEphemeralResource_Response{}
	renewResp := p.provider.RenewEphemeralResource(providers.RenewEphemeralResourceRequest{
		TypeName: req.TypeName,
		Private:  req.Private,
	})
	resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, renewResp.Diagnostics)
	if renewResp.Diagnostics.HasErrors() {
		return resp, nil
	}

	resp.Private = renewResp.Private
	resp.RenewAt = timestamppb.New(renewResp.RenewAt)
	return resp, nil
}

func (p *provider6) CloseEphemeralResource(_ context.Context, req *tfplugin6.CloseEphemeralResource_Request) (*tfplugin6.CloseEphemeralResource_Response, error) {
	resp := &tfplugin6.CloseEphemeralResource_Response{}
	closeResp := p.provider.CloseEphemeralResource(providers.CloseEphemeralResourceRequest{
		TypeName: req.TypeName,
		Private:  req.Private,
	})
	resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, closeResp.Diagnostics)
	if closeResp.Diagnostics.HasErrors() {
		return resp, nil
	}

	return resp, nil
}

func (p *provider6) GetFunctions(context.Context, *tfplugin6.GetFunctions_Request) (*tfplugin6.GetFunctions_Response, error) {
	panic("unimplemented")
}

func (p *provider6) CallFunction(_ context.Context, req *tfplugin6.CallFunction_Request) (*tfplugin6.CallFunction_Response, error) {
	var err error
	resp := &tfplugin6.CallFunction_Response{}

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
					resp.Error = &tfplugin6.FunctionError{
						Text:             "too many arguments for non-variadic function",
						FunctionArgument: &idx,
					}
					return resp, nil
				}
				argTy = funcSchema.VariadicParameter.Type
			}

			argVal, err := decodeDynamicValue6(rawArg, argTy)
			if err != nil {
				resp.Error = &tfplugin6.FunctionError{
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
		resp.Error = &tfplugin6.FunctionError{
			Text: callResp.Err.Error(),
		}

		if argErr, ok := callResp.Err.(function.ArgError); ok {
			idx := int64(argErr.Index)
			resp.Error.FunctionArgument = &idx
		}

		return resp, nil
	}

	resp.Result, err = encodeDynamicValue6(callResp.Result, funcSchema.ReturnType)
	if err != nil {
		resp.Error = &tfplugin6.FunctionError{
			Text: err.Error(),
		}

		return resp, nil
	}

	return resp, nil
}

func (p *provider6) GetResourceIdentitySchemas(_ context.Context, req *tfplugin6.GetResourceIdentitySchemas_Request) (*tfplugin6.GetResourceIdentitySchemas_Response, error) {
	resp := &tfplugin6.GetResourceIdentitySchemas_Response{
		IdentitySchemas: map[string]*tfplugin6.ResourceIdentitySchema{},
		Diagnostics:     []*tfplugin6.Diagnostic{},
	}

	for name, schema := range p.identitySchemas.IdentityTypes {
		resp.IdentitySchemas[name] = convert.ResourceIdentitySchemaToProto(schema)
	}

	resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, p.identitySchemas.Diagnostics)
	return resp, nil
}

func (p *provider6) UpgradeResourceIdentity(_ context.Context, req *tfplugin6.UpgradeResourceIdentity_Request) (*tfplugin6.UpgradeResourceIdentity_Response, error) {
	resp := &tfplugin6.UpgradeResourceIdentity_Response{}
	resource, ok := p.identitySchemas.IdentityTypes[req.TypeName]
	if !ok {
		return nil, fmt.Errorf("resource identity schema not found for type %q", req.TypeName)
	}
	ty := resource.Body.ImpliedType()

	upgradeResp := p.provider.UpgradeResourceIdentity(providers.UpgradeResourceIdentityRequest{
		TypeName:        req.TypeName,
		Version:         req.Version,
		RawIdentityJSON: req.RawIdentity.Json,
	})
	resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, upgradeResp.Diagnostics)

	if upgradeResp.Diagnostics.HasErrors() {
		return resp, nil
	}

	dv, err := encodeDynamicValue6(upgradeResp.UpgradedIdentity, ty)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	resp.UpgradedIdentity = &tfplugin6.ResourceIdentityData{
		IdentityData: dv,
	}
	return resp, nil
}

func (p *provider6) ListResource(req *tfplugin6.ListResource_Request, res tfplugin6.Provider_ListResourceServer) error {
	resourceSchema, ok := p.schema.ResourceTypes[req.TypeName]
	if !ok {
		return fmt.Errorf("resource schema not found for type %q", req.TypeName)
	}
	listSchema, ok := p.schema.ListResourceTypes[req.TypeName]
	if !ok {
		return fmt.Errorf("list resource schema not found for type %q", req.TypeName)
	}
	cfg, err := decodeDynamicValue6(req.Config, listSchema.Body.ImpliedType())
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
		var stateVal *tfplugin6.DynamicValue
		if item.Type().HasAttribute("state") {
			state := item.GetAttr("state")
			var err error
			stateVal, err = encodeDynamicValue6(state, resourceSchema.Body.ImpliedType())
			if err != nil {
				return status.Errorf(codes.Internal, "failed to encode list resource item state: %v", err)
			}
		}
		if !item.Type().HasAttribute("identity") {
			return status.Errorf(codes.Internal, "list resource item missing 'identity' attribute")
		}
		identity := item.GetAttr("identity")
		var identityVal *tfplugin6.DynamicValue
		identityVal, err = encodeDynamicValue6(identity, resourceSchema.Identity.ImpliedType())
		if err != nil {
			return status.Errorf(codes.Internal, "failed to encode list resource item identity: %v", err)
		}

		var displayName string
		if item.Type().HasAttribute("display_name") {
			displayName = item.GetAttr("display_name").AsString()
		}

		res.Send(&tfplugin6.ListResource_Event{
			Identity:       &tfplugin6.ResourceIdentityData{IdentityData: identityVal},
			ResourceObject: stateVal,
			DisplayName:    displayName,
		})
	}

	return nil
}

func (p *provider6) ValidateStateStoreConfig(ctx context.Context, req *tfplugin6.ValidateStateStore_Request) (*tfplugin6.ValidateStateStore_Response, error) {
	panic("not implemented")
}

func (p *provider6) ConfigureStateStore(ctx context.Context, req *tfplugin6.ConfigureStateStore_Request) (*tfplugin6.ConfigureStateStore_Response, error) {
	panic("not implemented")
}

func (p *provider6) GetStates(ctx context.Context, req *tfplugin6.GetStates_Request) (*tfplugin6.GetStates_Response, error) {
	panic("not implemented")
}

func (p *provider6) DeleteState(ctx context.Context, req *tfplugin6.DeleteState_Request) (*tfplugin6.DeleteState_Response, error) {
	panic("not implemented")
}

func (p *provider6) PlanAction(_ context.Context, req *tfplugin6.PlanAction_Request) (*tfplugin6.PlanAction_Response, error) {
	resp := &tfplugin6.PlanAction_Response{}

	actionSchema, ok := p.schema.Actions[req.ActionType]
	if !ok {
		return nil, fmt.Errorf("action schema not found for action %q", req.ActionType)
	}

	ty := actionSchema.ConfigSchema.ImpliedType()
	configVal, err := decodeDynamicValue6(req.Config, ty)
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

		priorState, err := decodeDynamicValue6(lr.PriorState, linkedResourceTy)
		if err != nil {
			return nil, fmt.Errorf("failed to decode prior state for linked resource #%d (%q) in action %q: %w", i, linkedResouceSchemas[i].TypeName, req.ActionType, err)
		}

		config, err := decodeDynamicValue6(lr.Config, linkedResourceTy)
		if err != nil {
			return nil, fmt.Errorf("failed to decode config for linked resource #%d (%q) in action %q: %w", i, linkedResouceSchemas[i].TypeName, req.ActionType, err)
		}

		var priorIdentity cty.Value
		if lr.PriorIdentity != nil && lr.PriorIdentity.IdentityData != nil {
			priorIdentity, err = decodeDynamicValue6(lr.PriorIdentity.IdentityData, linkedResourceIdentityTy)
			if err != nil {
				return nil, fmt.Errorf("failed to decode prior identity for linked resource #%d (%q) in action %q: %w", i, linkedResouceSchemas[i].TypeName, req.ActionType, err)
			}
		} else {
			priorIdentity = cty.NullVal(linkedResourceIdentityTy)
		}

		plannedState, err := decodeDynamicValue6(lr.PlannedState, linkedResourceTy)
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

	linkedResources := make([]*tfplugin6.PlanAction_Response_LinkedResource, 0, len(planResp.LinkedResources))
	for _, linked := range planResp.LinkedResources {
		plannedState, err := encodeDynamicValue6(linked.PlannedState, linked.PlannedState.Type())
		if err != nil {
			resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
			continue
		}

		plannedIdentity, err := encodeDynamicValue6(linked.PlannedIdentity, linked.PlannedIdentity.Type())
		if err != nil {
			resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
			continue
		}
		linkedResources = append(linkedResources, &tfplugin6.PlanAction_Response_LinkedResource{
			PlannedState: plannedState,
			PlannedIdentity: &tfplugin6.ResourceIdentityData{
				IdentityData: plannedIdentity,
			},
		})
	}
	resp.LinkedResources = linkedResources

	return resp, nil
}

func (p *provider6) InvokeAction(req *tfplugin6.InvokeAction_Request, server tfplugin6.Provider_InvokeActionServer) error {

	actionSchema, ok := p.schema.Actions[req.ActionType]
	if !ok {
		return fmt.Errorf("action schema not found for action %q", req.ActionType)
	}

	ty := actionSchema.ConfigSchema.ImpliedType()
	configVal, err := decodeDynamicValue6(req.Config, ty)
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

		priorState, err := decodeDynamicValue6(lr.PriorState, linkedResourceTy)
		if err != nil {
			return fmt.Errorf("failed to decode prior state for linked resource #%d (%q) in action %q: %w", i, linkedResourceSchemas[i].TypeName, req.ActionType, err)
		}

		plannedState, err := decodeDynamicValue6(lr.PlannedState, linkedResourceTy)
		if err != nil {
			return fmt.Errorf("failed to decode planned state for linked resource #%d (%q) in action %q: %w", i, linkedResourceSchemas[i].TypeName, req.ActionType, err)
		}

		config, err := decodeDynamicValue6(lr.Config, linkedResourceTy)
		if err != nil {
			return fmt.Errorf("failed to decode config for linked resource #%d (%q) in action %q: %w", i, linkedResourceSchemas[i].TypeName, req.ActionType, err)
		}

		plannedIdentity := cty.NullVal(linkedResourceIdentityTy)
		if lr.PlannedIdentity != nil && lr.PlannedIdentity.IdentityData != nil {
			plannedIdentity, err = decodeDynamicValue6(lr.PlannedIdentity.IdentityData, linkedResourceIdentityTy)
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
			server.Send(&tfplugin6.InvokeAction_Event{
				Type: &tfplugin6.InvokeAction_Event_Progress_{
					Progress: &tfplugin6.InvokeAction_Event_Progress{
						Message: invokeEvt.Message,
					},
				},
			})

		case providers.InvokeActionEvent_Completed:
			completed := &tfplugin6.InvokeAction_Event_Completed{
				LinkedResources: []*tfplugin6.InvokeAction_Event_Completed_LinkedResource{},
			}
			completed.Diagnostics = convert.AppendProtoDiag(completed.Diagnostics, invokeEvt.Diagnostics)

			for _, lr := range invokeEvt.LinkedResources {
				newState, err := encodeDynamicValue6(lr.NewState, lr.NewState.Type())
				if err != nil {
					return fmt.Errorf("failed to encode new state for linked resource: %w", err)
				}

				newIdentity, err := encodeDynamicValue6(lr.NewIdentity, lr.NewIdentity.Type())
				if err != nil {
					return fmt.Errorf("failed to encode new identity for linked resource: %w", err)

				}

				completed.LinkedResources = append(completed.LinkedResources, &tfplugin6.InvokeAction_Event_Completed_LinkedResource{
					NewState: newState,
					NewIdentity: &tfplugin6.ResourceIdentityData{
						IdentityData: newIdentity,
					},
					RequiresReplace: lr.RequiresReplace,
				})
			}

			err := server.Send(&tfplugin6.InvokeAction_Event{
				Type: &tfplugin6.InvokeAction_Event_Completed_{
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

func (p *provider6) StopProvider(context.Context, *tfplugin6.StopProvider_Request) (*tfplugin6.StopProvider_Response, error) {
	resp := &tfplugin6.StopProvider_Response{}
	err := p.provider.Stop()
	if err != nil {
		resp.Error = err.Error()
	}
	return resp, nil
}

// decode a DynamicValue from either the JSON or MsgPack encoding.
func decodeDynamicValue6(v *tfplugin6.DynamicValue, ty cty.Type) (cty.Value, error) {
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
func encodeDynamicValue6(v cty.Value, ty cty.Type) (*tfplugin6.DynamicValue, error) {
	mp, err := msgpack.Marshal(v, ty)
	return &tfplugin6.DynamicValue{
		Msgpack: mp,
	}, err
}
