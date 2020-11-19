package grpcwrap

import (
	"context"

	"github.com/hashicorp/terraform/internal/tfplugin5"
	"github.com/hashicorp/terraform/plugin/convert"
	"github.com/hashicorp/terraform/providers"
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"
	"github.com/zclconf/go-cty/cty/msgpack"
)

// New wraps a providers.Interface to implement a grpc ProviderServer.
// This is useful for creating a test binary out of an internal provider
// implementation.
func New(p providers.Interface) tfplugin5.ProviderServer {
	return &wrapped{
		provider: p,
		schema:   p.GetSchema(),
	}
}

type wrapped struct {
	provider providers.Interface
	schema   providers.GetSchemaResponse
}

func (w *wrapped) GetSchema(_ context.Context, req *tfplugin5.GetProviderSchema_Request) (*tfplugin5.GetProviderSchema_Response, error) {
	resp := &tfplugin5.GetProviderSchema_Response{
		ResourceSchemas:   make(map[string]*tfplugin5.Schema),
		DataSourceSchemas: make(map[string]*tfplugin5.Schema),
	}

	resp.Provider = &tfplugin5.Schema{
		Block: &tfplugin5.Schema_Block{},
	}
	if w.schema.Provider.Block != nil {
		resp.Provider.Block = convert.ConfigSchemaToProto(w.schema.Provider.Block)
	}

	resp.ProviderMeta = &tfplugin5.Schema{
		Block: &tfplugin5.Schema_Block{},
	}
	if w.schema.ProviderMeta.Block != nil {
		resp.ProviderMeta.Block = convert.ConfigSchemaToProto(w.schema.ProviderMeta.Block)
	}

	for typ, res := range w.schema.ResourceTypes {
		resp.ResourceSchemas[typ] = &tfplugin5.Schema{
			Version: res.Version,
			Block:   convert.ConfigSchemaToProto(res.Block),
		}
	}
	for typ, dat := range w.schema.DataSources {
		resp.DataSourceSchemas[typ] = &tfplugin5.Schema{
			Version: dat.Version,
			Block:   convert.ConfigSchemaToProto(dat.Block),
		}
	}

	// include any diagnostics from the original GetSchema call
	resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, w.schema.Diagnostics)

	return resp, nil
}

func (w *wrapped) PrepareProviderConfig(_ context.Context, req *tfplugin5.PrepareProviderConfig_Request) (*tfplugin5.PrepareProviderConfig_Response, error) {
	resp := &tfplugin5.PrepareProviderConfig_Response{}
	ty := w.schema.Provider.Block.ImpliedType()

	configVal, err := decodeDynamicValue(req.Config, ty)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	prepareResp := w.provider.PrepareProviderConfig(providers.PrepareProviderConfigRequest{
		Config: configVal,
	})

	// the PreparedConfig value is no longer used
	resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, prepareResp.Diagnostics)
	return resp, nil
}

func (w *wrapped) ValidateResourceTypeConfig(_ context.Context, req *tfplugin5.ValidateResourceTypeConfig_Request) (*tfplugin5.ValidateResourceTypeConfig_Response, error) {
	resp := &tfplugin5.ValidateResourceTypeConfig_Response{}
	ty := w.schema.ResourceTypes[req.TypeName].Block.ImpliedType()

	configVal, err := decodeDynamicValue(req.Config, ty)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	validateResp := w.provider.ValidateResourceTypeConfig(providers.ValidateResourceTypeConfigRequest{
		TypeName: req.TypeName,
		Config:   configVal,
	})

	resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, validateResp.Diagnostics)
	return resp, nil
}

func (w *wrapped) ValidateDataSourceConfig(_ context.Context, req *tfplugin5.ValidateDataSourceConfig_Request) (*tfplugin5.ValidateDataSourceConfig_Response, error) {
	resp := &tfplugin5.ValidateDataSourceConfig_Response{}
	ty := w.schema.DataSources[req.TypeName].Block.ImpliedType()

	configVal, err := decodeDynamicValue(req.Config, ty)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	validateResp := w.provider.ValidateDataSourceConfig(providers.ValidateDataSourceConfigRequest{
		TypeName: req.TypeName,
		Config:   configVal,
	})

	resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, validateResp.Diagnostics)
	return resp, nil
}

func (w *wrapped) UpgradeResourceState(_ context.Context, req *tfplugin5.UpgradeResourceState_Request) (*tfplugin5.UpgradeResourceState_Response, error) {
	resp := &tfplugin5.UpgradeResourceState_Response{}
	ty := w.schema.ResourceTypes[req.TypeName].Block.ImpliedType()

	upgradeResp := w.provider.UpgradeResourceState(providers.UpgradeResourceStateRequest{
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

func (w *wrapped) Configure(_ context.Context, req *tfplugin5.Configure_Request) (*tfplugin5.Configure_Response, error) {
	resp := &tfplugin5.Configure_Response{}
	ty := w.schema.Provider.Block.ImpliedType()

	configVal, err := decodeDynamicValue(req.Config, ty)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	configureResp := w.provider.Configure(providers.ConfigureRequest{
		TerraformVersion: req.TerraformVersion,
		Config:           configVal,
	})

	resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, configureResp.Diagnostics)
	return resp, nil
}

func (w *wrapped) ReadResource(_ context.Context, req *tfplugin5.ReadResource_Request) (*tfplugin5.ReadResource_Response, error) {
	resp := &tfplugin5.ReadResource_Response{}
	ty := w.schema.ResourceTypes[req.TypeName].Block.ImpliedType()

	stateVal, err := decodeDynamicValue(req.CurrentState, ty)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	metaTy := w.schema.ProviderMeta.Block.ImpliedType()
	metaVal, err := decodeDynamicValue(req.ProviderMeta, metaTy)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	readResp := w.provider.ReadResource(providers.ReadResourceRequest{
		TypeName:     req.TypeName,
		PriorState:   stateVal,
		Private:      req.Private,
		ProviderMeta: metaVal,
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

	return resp, nil
}

func (w *wrapped) PlanResourceChange(_ context.Context, req *tfplugin5.PlanResourceChange_Request) (*tfplugin5.PlanResourceChange_Response, error) {
	resp := &tfplugin5.PlanResourceChange_Response{}
	ty := w.schema.ResourceTypes[req.TypeName].Block.ImpliedType()

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

	metaTy := w.schema.ProviderMeta.Block.ImpliedType()
	metaVal, err := decodeDynamicValue(req.ProviderMeta, metaTy)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	planResp := w.provider.PlanResourceChange(providers.PlanResourceChangeRequest{
		TypeName:         req.TypeName,
		PriorState:       priorStateVal,
		ProposedNewState: proposedStateVal,
		Config:           configVal,
		PriorPrivate:     req.PriorPrivate,
		ProviderMeta:     metaVal,
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

	return resp, nil
}

func (w *wrapped) ApplyResourceChange(_ context.Context, req *tfplugin5.ApplyResourceChange_Request) (*tfplugin5.ApplyResourceChange_Response, error) {
	resp := &tfplugin5.ApplyResourceChange_Response{}
	ty := w.schema.ResourceTypes[req.TypeName].Block.ImpliedType()

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

	metaTy := w.schema.ProviderMeta.Block.ImpliedType()
	metaVal, err := decodeDynamicValue(req.ProviderMeta, metaTy)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	applyResp := w.provider.ApplyResourceChange(providers.ApplyResourceChangeRequest{
		TypeName:       req.TypeName,
		PriorState:     priorStateVal,
		PlannedState:   plannedStateVal,
		Config:         configVal,
		PlannedPrivate: req.PlannedPrivate,
		ProviderMeta:   metaVal,
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

	return resp, nil
}

func (w *wrapped) ImportResourceState(_ context.Context, req *tfplugin5.ImportResourceState_Request) (*tfplugin5.ImportResourceState_Response, error) {
	resp := &tfplugin5.ImportResourceState_Response{}

	importResp := w.provider.ImportResourceState(providers.ImportResourceStateRequest{
		TypeName: req.TypeName,
		ID:       req.Id,
	})
	resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, importResp.Diagnostics)

	for _, res := range importResp.ImportedResources {
		ty := w.schema.ResourceTypes[res.TypeName].Block.ImpliedType()
		state, err := encodeDynamicValue(res.State, ty)
		if err != nil {
			resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
			continue
		}

		resp.ImportedResources = append(resp.ImportedResources, &tfplugin5.ImportResourceState_ImportedResource{
			TypeName: res.TypeName,
			State:    state,
			Private:  res.Private,
		})
	}

	return resp, nil
}

func (w *wrapped) ReadDataSource(_ context.Context, req *tfplugin5.ReadDataSource_Request) (*tfplugin5.ReadDataSource_Response, error) {
	resp := &tfplugin5.ReadDataSource_Response{}
	ty := w.schema.DataSources[req.TypeName].Block.ImpliedType()

	configVal, err := decodeDynamicValue(req.Config, ty)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	metaTy := w.schema.ProviderMeta.Block.ImpliedType()
	metaVal, err := decodeDynamicValue(req.ProviderMeta, metaTy)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	readResp := w.provider.ReadDataSource(providers.ReadDataSourceRequest{
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

func (w *wrapped) Stop(context.Context, *tfplugin5.Stop_Request) (*tfplugin5.Stop_Response, error) {
	resp := &tfplugin5.Stop_Response{}
	err := w.provider.Stop()
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
