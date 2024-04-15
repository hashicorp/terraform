// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package plugin6

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/zclconf/go-cty/cty"

	plugin "github.com/hashicorp/go-plugin"
	"github.com/zclconf/go-cty/cty/function"
	ctyjson "github.com/zclconf/go-cty/cty/json"
	"github.com/zclconf/go-cty/cty/msgpack"
	"google.golang.org/grpc"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/logging"
	"github.com/hashicorp/terraform/internal/plugin6/convert"
	"github.com/hashicorp/terraform/internal/providers"
	proto6 "github.com/hashicorp/terraform/internal/tfplugin6"
)

var logger = logging.HCLogger()

// GRPCProviderPlugin implements plugin.GRPCPlugin for the go-plugin package.
type GRPCProviderPlugin struct {
	plugin.Plugin
	GRPCProvider func() proto6.ProviderServer
}

func (p *GRPCProviderPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &GRPCProvider{
		client: proto6.NewProviderClient(c),
		ctx:    ctx,
	}, nil
}

func (p *GRPCProviderPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	proto6.RegisterProviderServer(s, p.GRPCProvider())
	return nil
}

// GRPCProvider handles the client, or core side of the plugin rpc connection.
// The GRPCProvider methods are mostly a translation layer between the
// terraform providers types and the grpc proto types, directly converting
// between the two.
type GRPCProvider struct {
	// PluginClient provides a reference to the plugin.Client which controls the plugin process.
	// This allows the GRPCProvider a way to shutdown the plugin process.
	PluginClient *plugin.Client

	// TestServer contains a grpc.Server to close when the GRPCProvider is being
	// used in an end to end test of a provider.
	TestServer *grpc.Server

	// Addr uniquely identifies the type of provider.
	// Normally executed providers will have this set during initialization,
	// but it may not always be available for alternative execute modes.
	Addr addrs.Provider

	// Proto client use to make the grpc service calls.
	client proto6.ProviderClient

	// this context is created by the plugin package, and is canceled when the
	// plugin process ends.
	ctx context.Context

	// schema stores the schema for this provider. This is used to properly
	// serialize the requests for schemas.
	mu     sync.Mutex
	schema providers.GetProviderSchemaResponse
}

func (p *GRPCProvider) GetProviderSchema() providers.GetProviderSchemaResponse {
	p.mu.Lock()
	defer p.mu.Unlock()

	// check the global cache if we can
	// FIXME: A global cache is inappropriate when Terraform Core is being
	// used in a non-Terraform-CLI mode where we shouldn't assume that all
	// calls share the same provider implementations.
	if !p.Addr.IsZero() {
		if resp, ok := providers.SchemaCache.Get(p.Addr); ok && resp.ServerCapabilities.GetProviderSchemaOptional {
			logger.Trace("GRPCProvider.v6: returning cached schema", p.Addr.String())
			return resp
		}
	}
	logger.Trace("GRPCProvider.v6: GetProviderSchema")

	// If the local cache is non-zero, we know this instance has called
	// GetProviderSchema at least once and we can return early.
	if p.schema.Provider.Block != nil {
		return p.schema
	}

	var resp providers.GetProviderSchemaResponse

	resp.ResourceTypes = make(map[string]providers.Schema)
	resp.DataSources = make(map[string]providers.Schema)

	// Some providers may generate quite large schemas, and the internal default
	// grpc response size limit is 4MB. 64MB should cover most any use case, and
	// if we get providers nearing that we may want to consider a finer-grained
	// API to fetch individual resource schemas.
	// Note: this option is marked as EXPERIMENTAL in the grpc API. We keep
	// this for compatibility, but recent providers all set the max message
	// size much higher on the server side, which is the supported method for
	// determining payload size.
	const maxRecvSize = 64 << 20
	protoResp, err := p.client.GetProviderSchema(p.ctx, new(proto6.GetProviderSchema_Request), grpc.MaxRecvMsgSizeCallOption{MaxRecvMsgSize: maxRecvSize})
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(grpcErr(err))
		return resp
	}

	resp.Diagnostics = resp.Diagnostics.Append(convert.ProtoToDiagnostics(protoResp.Diagnostics))

	if resp.Diagnostics.HasErrors() {
		return resp
	}

	if protoResp.Provider == nil {
		resp.Diagnostics = resp.Diagnostics.Append(errors.New("missing provider schema"))
		return resp
	}

	resp.Provider = convert.ProtoToProviderSchema(protoResp.Provider)
	if protoResp.ProviderMeta == nil {
		logger.Debug("No provider meta schema returned")
	} else {
		resp.ProviderMeta = convert.ProtoToProviderSchema(protoResp.ProviderMeta)
	}

	for name, res := range protoResp.ResourceSchemas {
		resp.ResourceTypes[name] = convert.ProtoToProviderSchema(res)
	}

	for name, data := range protoResp.DataSourceSchemas {
		resp.DataSources[name] = convert.ProtoToProviderSchema(data)
	}

	if decls, err := convert.FunctionDeclsFromProto(protoResp.Functions); err == nil {
		resp.Functions = decls
	} else {
		resp.Diagnostics = resp.Diagnostics.Append(err)
		return resp
	}

	if protoResp.ServerCapabilities != nil {
		resp.ServerCapabilities.PlanDestroy = protoResp.ServerCapabilities.PlanDestroy
		resp.ServerCapabilities.GetProviderSchemaOptional = protoResp.ServerCapabilities.GetProviderSchemaOptional
		resp.ServerCapabilities.MoveResourceState = protoResp.ServerCapabilities.MoveResourceState
	}

	// set the global cache if we can
	// FIXME: A global cache is inappropriate when Terraform Core is being
	// used in a non-Terraform-CLI mode where we shouldn't assume that all
	// calls share the same provider implementations.
	if !p.Addr.IsZero() {
		providers.SchemaCache.Set(p.Addr, resp)
	}

	// always store this here in the client for providers that are not able to
	// use GetProviderSchemaOptional
	p.schema = resp

	return resp
}

func (p *GRPCProvider) ValidateProviderConfig(r providers.ValidateProviderConfigRequest) (resp providers.ValidateProviderConfigResponse) {
	logger.Trace("GRPCProvider.v6: ValidateProviderConfig")

	schema := p.GetProviderSchema()
	if schema.Diagnostics.HasErrors() {
		resp.Diagnostics = schema.Diagnostics
		return resp
	}

	ty := schema.Provider.Block.ImpliedType()

	mp, err := msgpack.Marshal(r.Config, ty)
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(err)
		return resp
	}

	protoReq := &proto6.ValidateProviderConfig_Request{
		Config: &proto6.DynamicValue{Msgpack: mp},
	}

	protoResp, err := p.client.ValidateProviderConfig(p.ctx, protoReq)
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(grpcErr(err))
		return resp
	}

	resp.Diagnostics = resp.Diagnostics.Append(convert.ProtoToDiagnostics(protoResp.Diagnostics))
	return resp
}

func (p *GRPCProvider) ValidateResourceConfig(r providers.ValidateResourceConfigRequest) (resp providers.ValidateResourceConfigResponse) {
	logger.Trace("GRPCProvider.v6: ValidateResourceConfig")

	schema := p.GetProviderSchema()
	if schema.Diagnostics.HasErrors() {
		resp.Diagnostics = schema.Diagnostics
		return resp
	}

	resourceSchema, ok := schema.ResourceTypes[r.TypeName]
	if !ok {
		resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("unknown resource type %q", r.TypeName))
		return resp
	}

	mp, err := msgpack.Marshal(r.Config, resourceSchema.Block.ImpliedType())
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(err)
		return resp
	}

	protoReq := &proto6.ValidateResourceConfig_Request{
		TypeName: r.TypeName,
		Config:   &proto6.DynamicValue{Msgpack: mp},
	}

	protoResp, err := p.client.ValidateResourceConfig(p.ctx, protoReq)
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(grpcErr(err))
		return resp
	}

	resp.Diagnostics = resp.Diagnostics.Append(convert.ProtoToDiagnostics(protoResp.Diagnostics))
	return resp
}

func (p *GRPCProvider) ValidateDataResourceConfig(r providers.ValidateDataResourceConfigRequest) (resp providers.ValidateDataResourceConfigResponse) {
	logger.Trace("GRPCProvider.v6: ValidateDataResourceConfig")

	schema := p.GetProviderSchema()
	if schema.Diagnostics.HasErrors() {
		resp.Diagnostics = schema.Diagnostics
		return resp
	}

	dataSchema, ok := schema.DataSources[r.TypeName]
	if !ok {
		resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("unknown data source %q", r.TypeName))
		return resp
	}

	mp, err := msgpack.Marshal(r.Config, dataSchema.Block.ImpliedType())
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(err)
		return resp
	}

	protoReq := &proto6.ValidateDataResourceConfig_Request{
		TypeName: r.TypeName,
		Config:   &proto6.DynamicValue{Msgpack: mp},
	}

	protoResp, err := p.client.ValidateDataResourceConfig(p.ctx, protoReq)
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(grpcErr(err))
		return resp
	}
	resp.Diagnostics = resp.Diagnostics.Append(convert.ProtoToDiagnostics(protoResp.Diagnostics))
	return resp
}

func (p *GRPCProvider) UpgradeResourceState(r providers.UpgradeResourceStateRequest) (resp providers.UpgradeResourceStateResponse) {
	logger.Trace("GRPCProvider.v6: UpgradeResourceState")

	schema := p.GetProviderSchema()
	if schema.Diagnostics.HasErrors() {
		resp.Diagnostics = schema.Diagnostics
		return resp
	}

	resSchema, ok := schema.ResourceTypes[r.TypeName]
	if !ok {
		resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("unknown resource type %q", r.TypeName))
		return resp
	}

	protoReq := &proto6.UpgradeResourceState_Request{
		TypeName: r.TypeName,
		Version:  int64(r.Version),
		RawState: &proto6.RawState{
			Json:    r.RawStateJSON,
			Flatmap: r.RawStateFlatmap,
		},
	}

	protoResp, err := p.client.UpgradeResourceState(p.ctx, protoReq)
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(grpcErr(err))
		return resp
	}
	resp.Diagnostics = resp.Diagnostics.Append(convert.ProtoToDiagnostics(protoResp.Diagnostics))

	ty := resSchema.Block.ImpliedType()
	resp.UpgradedState = cty.NullVal(ty)
	if protoResp.UpgradedState == nil {
		return resp
	}

	state, err := decodeDynamicValue(protoResp.UpgradedState, ty)
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(err)
		return resp
	}
	resp.UpgradedState = state

	return resp
}

func (p *GRPCProvider) ConfigureProvider(r providers.ConfigureProviderRequest) (resp providers.ConfigureProviderResponse) {
	logger.Trace("GRPCProvider.v6: ConfigureProvider")

	schema := p.GetProviderSchema()

	var mp []byte

	// we don't have anything to marshal if there's no config
	mp, err := msgpack.Marshal(r.Config, schema.Provider.Block.ImpliedType())
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(err)
		return resp
	}

	protoReq := &proto6.ConfigureProvider_Request{
		TerraformVersion: r.TerraformVersion,
		Config: &proto6.DynamicValue{
			Msgpack: mp,
		},
	}

	protoResp, err := p.client.ConfigureProvider(p.ctx, protoReq)
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(grpcErr(err))
		return resp
	}
	resp.Diagnostics = resp.Diagnostics.Append(convert.ProtoToDiagnostics(protoResp.Diagnostics))
	return resp
}

func (p *GRPCProvider) Stop() error {
	logger.Trace("GRPCProvider.v6: Stop")

	resp, err := p.client.StopProvider(p.ctx, new(proto6.StopProvider_Request))
	if err != nil {
		return err
	}

	if resp.Error != "" {
		return errors.New(resp.Error)
	}
	return nil
}

func (p *GRPCProvider) ReadResource(r providers.ReadResourceRequest) (resp providers.ReadResourceResponse) {
	logger.Trace("GRPCProvider.v6: ReadResource")

	schema := p.GetProviderSchema()
	if schema.Diagnostics.HasErrors() {
		resp.Diagnostics = schema.Diagnostics
		return resp
	}

	resSchema, ok := schema.ResourceTypes[r.TypeName]
	if !ok {
		resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("unknown resource type " + r.TypeName))
		return resp
	}

	metaSchema := schema.ProviderMeta

	mp, err := msgpack.Marshal(r.PriorState, resSchema.Block.ImpliedType())
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(err)
		return resp
	}

	protoReq := &proto6.ReadResource_Request{
		TypeName:        r.TypeName,
		CurrentState:    &proto6.DynamicValue{Msgpack: mp},
		Private:         r.Private,
		DeferralAllowed: r.DeferralAllowed,
	}

	if metaSchema.Block != nil {
		metaMP, err := msgpack.Marshal(r.ProviderMeta, metaSchema.Block.ImpliedType())
		if err != nil {
			resp.Diagnostics = resp.Diagnostics.Append(err)
			return resp
		}
		protoReq.ProviderMeta = &proto6.DynamicValue{Msgpack: metaMP}
	}

	protoResp, err := p.client.ReadResource(p.ctx, protoReq)
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(grpcErr(err))
		return resp
	}
	resp.Diagnostics = resp.Diagnostics.Append(convert.ProtoToDiagnostics(protoResp.Diagnostics))

	state, err := decodeDynamicValue(protoResp.NewState, resSchema.Block.ImpliedType())
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(err)
		return resp
	}
	resp.NewState = state
	resp.Private = protoResp.Private
	resp.Deferred = convert.ProtoToDeferred(protoResp.Deferred)

	return resp
}

func (p *GRPCProvider) PlanResourceChange(r providers.PlanResourceChangeRequest) (resp providers.PlanResourceChangeResponse) {
	logger.Trace("GRPCProvider.v6: PlanResourceChange")

	schema := p.GetProviderSchema()
	if schema.Diagnostics.HasErrors() {
		resp.Diagnostics = schema.Diagnostics
		return resp
	}

	resSchema, ok := schema.ResourceTypes[r.TypeName]
	if !ok {
		resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("unknown resource type %q", r.TypeName))
		return resp
	}

	metaSchema := schema.ProviderMeta
	capabilities := schema.ServerCapabilities

	// If the provider doesn't support planning a destroy operation, we can
	// return immediately.
	if r.ProposedNewState.IsNull() && !capabilities.PlanDestroy {
		resp.PlannedState = r.ProposedNewState
		resp.PlannedPrivate = r.PriorPrivate
		return resp
	}

	priorMP, err := msgpack.Marshal(r.PriorState, resSchema.Block.ImpliedType())
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(err)
		return resp
	}

	configMP, err := msgpack.Marshal(r.Config, resSchema.Block.ImpliedType())
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(err)
		return resp
	}

	propMP, err := msgpack.Marshal(r.ProposedNewState, resSchema.Block.ImpliedType())
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(err)
		return resp
	}

	protoReq := &proto6.PlanResourceChange_Request{
		TypeName:         r.TypeName,
		PriorState:       &proto6.DynamicValue{Msgpack: priorMP},
		Config:           &proto6.DynamicValue{Msgpack: configMP},
		ProposedNewState: &proto6.DynamicValue{Msgpack: propMP},
		PriorPrivate:     r.PriorPrivate,
	}

	if metaSchema.Block != nil {
		metaTy := metaSchema.Block.ImpliedType()
		metaVal := r.ProviderMeta
		if metaVal == cty.NilVal {
			metaVal = cty.NullVal(metaTy)
		}
		metaMP, err := msgpack.Marshal(metaVal, metaTy)
		if err != nil {
			resp.Diagnostics = resp.Diagnostics.Append(err)
			return resp
		}
		protoReq.ProviderMeta = &proto6.DynamicValue{Msgpack: metaMP}
	}

	protoResp, err := p.client.PlanResourceChange(p.ctx, protoReq)
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(grpcErr(err))
		return resp
	}
	resp.Diagnostics = resp.Diagnostics.Append(convert.ProtoToDiagnostics(protoResp.Diagnostics))

	state, err := decodeDynamicValue(protoResp.PlannedState, resSchema.Block.ImpliedType())
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(err)
		return resp
	}
	resp.PlannedState = state

	for _, p := range protoResp.RequiresReplace {
		resp.RequiresReplace = append(resp.RequiresReplace, convert.AttributePathToPath(p))
	}

	resp.PlannedPrivate = protoResp.PlannedPrivate

	resp.LegacyTypeSystem = protoResp.LegacyTypeSystem

	return resp
}

func (p *GRPCProvider) ApplyResourceChange(r providers.ApplyResourceChangeRequest) (resp providers.ApplyResourceChangeResponse) {
	logger.Trace("GRPCProvider.v6: ApplyResourceChange")

	schema := p.GetProviderSchema()
	if schema.Diagnostics.HasErrors() {
		resp.Diagnostics = schema.Diagnostics
		return resp
	}

	resSchema, ok := schema.ResourceTypes[r.TypeName]
	if !ok {
		resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("unknown resource type %q", r.TypeName))
		return resp
	}

	metaSchema := schema.ProviderMeta

	priorMP, err := msgpack.Marshal(r.PriorState, resSchema.Block.ImpliedType())
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(err)
		return resp
	}
	plannedMP, err := msgpack.Marshal(r.PlannedState, resSchema.Block.ImpliedType())
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(err)
		return resp
	}
	configMP, err := msgpack.Marshal(r.Config, resSchema.Block.ImpliedType())
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(err)
		return resp
	}

	protoReq := &proto6.ApplyResourceChange_Request{
		TypeName:       r.TypeName,
		PriorState:     &proto6.DynamicValue{Msgpack: priorMP},
		PlannedState:   &proto6.DynamicValue{Msgpack: plannedMP},
		Config:         &proto6.DynamicValue{Msgpack: configMP},
		PlannedPrivate: r.PlannedPrivate,
	}

	if metaSchema.Block != nil {
		metaTy := metaSchema.Block.ImpliedType()
		metaVal := r.ProviderMeta
		if metaVal == cty.NilVal {
			metaVal = cty.NullVal(metaTy)
		}
		metaMP, err := msgpack.Marshal(metaVal, metaTy)
		if err != nil {
			resp.Diagnostics = resp.Diagnostics.Append(err)
			return resp
		}
		protoReq.ProviderMeta = &proto6.DynamicValue{Msgpack: metaMP}
	}

	protoResp, err := p.client.ApplyResourceChange(p.ctx, protoReq)
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(grpcErr(err))
		return resp
	}
	resp.Diagnostics = resp.Diagnostics.Append(convert.ProtoToDiagnostics(protoResp.Diagnostics))

	resp.Private = protoResp.Private

	state, err := decodeDynamicValue(protoResp.NewState, resSchema.Block.ImpliedType())
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(err)
		return resp
	}
	resp.NewState = state

	resp.LegacyTypeSystem = protoResp.LegacyTypeSystem

	return resp
}

func (p *GRPCProvider) ImportResourceState(r providers.ImportResourceStateRequest) (resp providers.ImportResourceStateResponse) {
	logger.Trace("GRPCProvider.v6: ImportResourceState")

	schema := p.GetProviderSchema()
	if schema.Diagnostics.HasErrors() {
		resp.Diagnostics = schema.Diagnostics
		return resp
	}

	protoReq := &proto6.ImportResourceState_Request{
		TypeName: r.TypeName,
		Id:       r.ID,
	}

	protoResp, err := p.client.ImportResourceState(p.ctx, protoReq)
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(grpcErr(err))
		return resp
	}
	resp.Diagnostics = resp.Diagnostics.Append(convert.ProtoToDiagnostics(protoResp.Diagnostics))

	for _, imported := range protoResp.ImportedResources {
		resource := providers.ImportedResource{
			TypeName: imported.TypeName,
			Private:  imported.Private,
		}

		resSchema, ok := schema.ResourceTypes[r.TypeName]
		if !ok {
			resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("unknown resource type %q", r.TypeName))
			continue
		}

		state, err := decodeDynamicValue(imported.State, resSchema.Block.ImpliedType())
		if err != nil {
			resp.Diagnostics = resp.Diagnostics.Append(err)
			return resp
		}
		resource.State = state
		resp.ImportedResources = append(resp.ImportedResources, resource)
	}

	return resp
}

func (p *GRPCProvider) MoveResourceState(r providers.MoveResourceStateRequest) (resp providers.MoveResourceStateResponse) {
	logger.Trace("GRPCProvider: MoveResourceState")

	protoReq := &proto6.MoveResourceState_Request{
		SourceProviderAddress: r.SourceProviderAddress,
		SourceTypeName:        r.SourceTypeName,
		SourceSchemaVersion:   r.SourceSchemaVersion,
		SourceState: &proto6.RawState{
			Json: r.SourceStateJSON,
		},
		SourcePrivate:  r.SourcePrivate,
		TargetTypeName: r.TargetTypeName,
	}

	protoResp, err := p.client.MoveResourceState(p.ctx, protoReq)
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(grpcErr(err))
		return resp
	}
	resp.Diagnostics = resp.Diagnostics.Append(convert.ProtoToDiagnostics(protoResp.Diagnostics))
	if resp.Diagnostics.HasErrors() {
		return resp
	}

	schema := p.GetProviderSchema()
	if schema.Diagnostics.HasErrors() {
		resp.Diagnostics = schema.Diagnostics
		return resp
	}

	targetType, ok := schema.ResourceTypes[r.TargetTypeName]
	if !ok {
		// We should have validated this earlier in the process, but we'll
		// still return an error instead of crashing in case something went
		// wrong.
		resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("unknown resource type %q; this is a bug in Terraform - please report it", r.TargetTypeName))
		return resp
	}
	resp.TargetState, err = decodeDynamicValue(protoResp.TargetState, targetType.Block.ImpliedType())
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(err)
		return resp
	}

	resp.TargetPrivate = protoResp.TargetPrivate

	return resp
}

func (p *GRPCProvider) ReadDataSource(r providers.ReadDataSourceRequest) (resp providers.ReadDataSourceResponse) {
	logger.Trace("GRPCProvider.v6: ReadDataSource")

	schema := p.GetProviderSchema()
	if schema.Diagnostics.HasErrors() {
		resp.Diagnostics = schema.Diagnostics
		return resp
	}

	dataSchema, ok := schema.DataSources[r.TypeName]
	if !ok {
		schema.Diagnostics = schema.Diagnostics.Append(fmt.Errorf("unknown data source %q", r.TypeName))
	}

	metaSchema := schema.ProviderMeta

	config, err := msgpack.Marshal(r.Config, dataSchema.Block.ImpliedType())
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(err)
		return resp
	}

	protoReq := &proto6.ReadDataSource_Request{
		TypeName: r.TypeName,
		Config: &proto6.DynamicValue{
			Msgpack: config,
		},
		DeferralAllowed: r.DeferralAllowed,
	}

	if metaSchema.Block != nil {
		metaMP, err := msgpack.Marshal(r.ProviderMeta, metaSchema.Block.ImpliedType())
		if err != nil {
			resp.Diagnostics = resp.Diagnostics.Append(err)
			return resp
		}
		protoReq.ProviderMeta = &proto6.DynamicValue{Msgpack: metaMP}
	}

	protoResp, err := p.client.ReadDataSource(p.ctx, protoReq)
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(grpcErr(err))
		return resp
	}
	resp.Diagnostics = resp.Diagnostics.Append(convert.ProtoToDiagnostics(protoResp.Diagnostics))

	state, err := decodeDynamicValue(protoResp.State, dataSchema.Block.ImpliedType())
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(err)
		return resp
	}
	resp.State = state
	resp.Deferred = convert.ProtoToDeferred(protoResp.Deferred)

	return resp
}

func (p *GRPCProvider) CallFunction(r providers.CallFunctionRequest) (resp providers.CallFunctionResponse) {
	logger.Trace("GRPCProvider.v6", "CallFunction", r.FunctionName)

	schema := p.GetProviderSchema()
	if schema.Diagnostics.HasErrors() {
		resp.Err = schema.Diagnostics.Err()
		return resp
	}

	funcDecl, ok := schema.Functions[r.FunctionName]
	// We check for various problems with the request below in the interests
	// of robustness, just to avoid crashing while trying to encode/decode, but
	// if we reach any of these errors then that suggests a bug in the caller,
	// because we should catch function calls that don't match the schema at an
	// earlier point than this.
	if !ok {
		// Should only get here if the caller has a bug, because we should
		// have detected earlier any attempt to call a function that the
		// provider didn't declare.
		resp.Err = fmt.Errorf("provider has no function named %q", r.FunctionName)
		return resp
	}
	if len(r.Arguments) < len(funcDecl.Parameters) {
		resp.Err = fmt.Errorf("not enough arguments for function %q", r.FunctionName)
		return resp
	}
	if funcDecl.VariadicParameter == nil && len(r.Arguments) > len(funcDecl.Parameters) {
		resp.Err = fmt.Errorf("too many arguments for function %q", r.FunctionName)
		return resp
	}
	args := make([]*proto6.DynamicValue, len(r.Arguments))
	for i, argVal := range r.Arguments {
		var paramDecl providers.FunctionParam
		if i < len(funcDecl.Parameters) {
			paramDecl = funcDecl.Parameters[i]
		} else {
			paramDecl = *funcDecl.VariadicParameter
		}

		argValRaw, err := msgpack.Marshal(argVal, paramDecl.Type)
		if err != nil {
			resp.Err = err
			return resp
		}
		args[i] = &proto6.DynamicValue{
			Msgpack: argValRaw,
		}
	}

	protoResp, err := p.client.CallFunction(p.ctx, &proto6.CallFunction_Request{
		Name:      r.FunctionName,
		Arguments: args,
	})
	if err != nil {
		// functions can only support simple errors, but use our grpcError
		// diagnostic function to format common problems is a more
		// user-friendly manner.
		resp.Err = grpcErr(err).Err()
		return resp
	}

	if protoResp.Error != nil {
		resp.Err = errors.New(protoResp.Error.Text)

		// If this is a problem with a specific argument, we can wrap the error
		// in a function.ArgError
		if protoResp.Error.FunctionArgument != nil {
			resp.Err = function.NewArgError(int(*protoResp.Error.FunctionArgument), resp.Err)
		}

		return resp
	}

	resultVal, err := decodeDynamicValue(protoResp.Result, funcDecl.ReturnType)
	if err != nil {
		resp.Err = err
		return resp
	}

	resp.Result = resultVal
	return resp
}

// closing the grpc connection is final, and terraform will call it at the end of every phase.
func (p *GRPCProvider) Close() error {
	logger.Trace("GRPCProvider.v6: Close")

	// Make sure to stop the server if we're not running within go-plugin.
	if p.TestServer != nil {
		p.TestServer.Stop()
	}

	// Check this since it's not automatically inserted during plugin creation.
	// It's currently only inserted by the command package, because that is
	// where the factory is built and is the only point with access to the
	// plugin.Client.
	if p.PluginClient == nil {
		logger.Debug("provider has no plugin.Client")
		return nil
	}

	p.PluginClient.Kill()
	return nil
}

// Decode a DynamicValue from either the JSON or MsgPack encoding.
func decodeDynamicValue(v *proto6.DynamicValue, ty cty.Type) (cty.Value, error) {
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
