package plugin

import (
	"context"
	"errors"
	"log"
	"sync"

	"github.com/zclconf/go-cty/cty"

	plugin "github.com/hashicorp/go-plugin"
	"github.com/hashicorp/terraform/plugin/convert"
	"github.com/hashicorp/terraform/plugin/proto"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/version"
	"github.com/zclconf/go-cty/cty/msgpack"
	"google.golang.org/grpc"
)

// GRPCProviderPlugin implements plugin.GRPCPlugin for the go-plugin package.
type GRPCProviderPlugin struct {
	plugin.Plugin
	GRPCProvider func() proto.ProviderServer
}

func (p *GRPCProviderPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &GRPCProvider{
		client: proto.NewProviderClient(c),
		ctx:    ctx,
	}, nil
}

func (p *GRPCProviderPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	proto.RegisterProviderServer(s, p.GRPCProvider())
	return nil
}

// GRPCProvider handles the client, or core side of the plugin rpc connection.
// The GRPCProvider methods are mostly a translation layer between the
// terraform provioders types and the grpc proto types, directly converting
// between the two.
type GRPCProvider struct {
	// PluginClient provides a reference to the plugin.Client which controls the plugin process.
	// This allows the GRPCProvider a way to shutdown the plugin process.
	PluginClient *plugin.Client

	// Proto client use to make the grpc service calls.
	client proto.ProviderClient

	// this context is created by the plugin package, and is canceled when the
	// plugin process ends.
	ctx context.Context

	// schema stores the schema for this provider. This is used to properly
	// serialize the state for requests.
	mu      sync.Mutex
	schemas providers.GetSchemaResponse
}

// getSchema is used internally to get the saved provider schema.  The schema
// should have already been fetched from the provider, but we have to
// synchronize access to avoid being called concurrently with GetSchema.
func (p *GRPCProvider) getSchema() providers.GetSchemaResponse {
	p.mu.Lock()
	// unlock inline in case GetSchema needs to be called
	if p.schemas.Provider.Block != nil {
		p.mu.Unlock()
		return p.schemas
	}
	p.mu.Unlock()

	// the schema should have been fetched already, but give it another shot
	// just in case things are being called out of order. This may happen for
	// tests.
	schemas := p.GetSchema()
	if schemas.Diagnostics.HasErrors() {
		panic(schemas.Diagnostics.Err())
	}

	return schemas
}

// getResourceSchema is a helper to extract the schema for a resource, and
// panics if the schema is not available.
func (p *GRPCProvider) getResourceSchema(name string) providers.Schema {
	schema := p.getSchema()
	resSchema, ok := schema.ResourceTypes[name]
	if !ok {
		panic("unknown resource type " + name)
	}
	return resSchema
}

// gettDatasourceSchema is a helper to extract the schema for a datasource, and
// panics if that schema is not available.
func (p *GRPCProvider) getDatasourceSchema(name string) providers.Schema {
	schema := p.getSchema()
	dataSchema, ok := schema.DataSources[name]
	if !ok {
		panic("unknown data source " + name)
	}
	return dataSchema
}

func (p *GRPCProvider) GetSchema() (resp providers.GetSchemaResponse) {
	log.Printf("[TRACE] GRPCProvider: GetSchema")
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.schemas.Provider.Block != nil {
		return p.schemas
	}

	resp.ResourceTypes = make(map[string]providers.Schema)
	resp.DataSources = make(map[string]providers.Schema)

	protoResp, err := p.client.GetSchema(p.ctx, new(proto.GetProviderSchema_Request))
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(err)
		return resp
	}

	if protoResp.Provider == nil {
		resp.Diagnostics = resp.Diagnostics.Append(errors.New("missing provider schema"))
		return resp
	}

	resp.Provider = convert.ProtoToProviderSchema(protoResp.Provider)

	for name, res := range protoResp.ResourceSchemas {
		resp.ResourceTypes[name] = convert.ProtoToProviderSchema(res)
	}

	for name, data := range protoResp.DataSourceSchemas {
		resp.DataSources[name] = convert.ProtoToProviderSchema(data)
	}

	p.schemas = resp

	return resp
}

func (p *GRPCProvider) ValidateProviderConfig(r providers.ValidateProviderConfigRequest) (resp providers.ValidateProviderConfigResponse) {
	log.Printf("[TRACE] GRPCProvider: ValidateProviderConfig")

	schema := p.getSchema()
	mp, err := msgpack.Marshal(r.Config, schema.Provider.Block.ImpliedType())
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(err)
		return resp
	}

	protoReq := &proto.ValidateProviderConfig_Request{
		Config: &proto.DynamicValue{Msgpack: mp},
	}

	protoResp, err := p.client.ValidateProviderConfig(p.ctx, protoReq)
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(err)
		return resp
	}
	resp.Diagnostics = resp.Diagnostics.Append(convert.ProtoToDiagnostics(protoResp.Diagnostics))
	return resp
}

func (p *GRPCProvider) ValidateResourceTypeConfig(r providers.ValidateResourceTypeConfigRequest) (resp providers.ValidateResourceTypeConfigResponse) {
	log.Printf("[TRACE] GRPCProvider: ValidateResourceTypeConfig")

	resourceSchema := p.getResourceSchema(r.TypeName)

	mp, err := msgpack.Marshal(r.Config, resourceSchema.Block.ImpliedType())
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(err)
		return resp
	}

	protoReq := &proto.ValidateResourceTypeConfig_Request{
		TypeName: r.TypeName,
		Config:   &proto.DynamicValue{Msgpack: mp},
	}

	protoResp, err := p.client.ValidateResourceTypeConfig(p.ctx, protoReq)
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(err)
		return resp
	}

	resp.Diagnostics = resp.Diagnostics.Append(convert.ProtoToDiagnostics(protoResp.Diagnostics))
	return resp
}

func (p *GRPCProvider) ValidateDataSourceConfig(r providers.ValidateDataSourceConfigRequest) (resp providers.ValidateDataSourceConfigResponse) {
	log.Printf("[TRACE] GRPCProvider: ValidateDataSourceConfig")

	dataSchema := p.getDatasourceSchema(r.TypeName)

	mp, err := msgpack.Marshal(r.Config, dataSchema.Block.ImpliedType())
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(err)
		return resp
	}

	protoReq := &proto.ValidateDataSourceConfig_Request{
		TypeName: r.TypeName,
		Config:   &proto.DynamicValue{Msgpack: mp},
	}

	protoResp, err := p.client.ValidateDataSourceConfig(p.ctx, protoReq)
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(err)
		return resp
	}
	resp.Diagnostics = resp.Diagnostics.Append(convert.ProtoToDiagnostics(protoResp.Diagnostics))
	return resp
}

func (p *GRPCProvider) UpgradeResourceState(r providers.UpgradeResourceStateRequest) (resp providers.UpgradeResourceStateResponse) {
	log.Printf("[TRACE] GRPCProvider: UpgradeResourceState")

	resSchema := p.getResourceSchema(r.TypeName)

	protoReq := &proto.UpgradeResourceState_Request{
		TypeName: r.TypeName,
		Version:  int64(r.Version),
		RawState: &proto.RawState{
			Json:    r.RawStateJSON,
			Flatmap: r.RawStateFlatmap,
		},
	}

	protoResp, err := p.client.UpgradeResourceState(p.ctx, protoReq)
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(err)
		return resp
	}

	state := cty.NullVal(resSchema.Block.ImpliedType())
	if protoResp.UpgradedState != nil {
		state, err = msgpack.Unmarshal(protoResp.UpgradedState.Msgpack, resSchema.Block.ImpliedType())
		if err != nil {
			resp.Diagnostics = resp.Diagnostics.Append(err)
			return resp
		}
	}

	resp.UpgradedState = state

	resp.Diagnostics = resp.Diagnostics.Append(convert.ProtoToDiagnostics(protoResp.Diagnostics))
	return resp
}

func (p *GRPCProvider) Configure(r providers.ConfigureRequest) (resp providers.ConfigureResponse) {
	log.Printf("[TRACE] GRPCProvider: Configure")

	schema := p.getSchema()

	var mp []byte

	// we don't have anything to marshal if there's no config
	mp, err := msgpack.Marshal(r.Config, schema.Provider.Block.ImpliedType())
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(err)
		return resp
	}

	protoReq := &proto.Configure_Request{
		TerraformVersion: version.Version,
		Config: &proto.DynamicValue{
			Msgpack: mp,
		},
	}

	protoResp, err := p.client.Configure(p.ctx, protoReq)
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(err)
		return resp
	}
	resp.Diagnostics = resp.Diagnostics.Append(convert.ProtoToDiagnostics(protoResp.Diagnostics))
	return resp
}

func (p *GRPCProvider) Stop() error {
	log.Printf("[TRACE] GRPCProvider: Stop")

	resp, err := p.client.Stop(p.ctx, new(proto.Stop_Request))
	if err != nil {
		return err
	}

	if resp.Error != "" {
		return errors.New(resp.Error)
	}
	return nil
}

func (p *GRPCProvider) ReadResource(r providers.ReadResourceRequest) (resp providers.ReadResourceResponse) {
	log.Printf("[TRACE] GRPCProvider: ReadResource")

	resSchema := p.getResourceSchema(r.TypeName)

	mp, err := msgpack.Marshal(r.PriorState, resSchema.Block.ImpliedType())
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(err)
		return resp
	}

	protoReq := &proto.ReadResource_Request{
		TypeName:     r.TypeName,
		CurrentState: &proto.DynamicValue{Msgpack: mp},
	}

	protoResp, err := p.client.ReadResource(p.ctx, protoReq)
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(err)
		return resp
	}

	resp.Diagnostics = resp.Diagnostics.Append(convert.ProtoToDiagnostics(protoResp.Diagnostics))

	state := cty.NullVal(resSchema.Block.ImpliedType())
	if protoResp.NewState != nil {
		state, err = msgpack.Unmarshal(protoResp.NewState.Msgpack, resSchema.Block.ImpliedType())
		if err != nil {
			resp.Diagnostics = resp.Diagnostics.Append(err)
			return resp
		}
	}
	resp.NewState = state

	return resp
}

func (p *GRPCProvider) PlanResourceChange(r providers.PlanResourceChangeRequest) (resp providers.PlanResourceChangeResponse) {
	log.Printf("[TRACE] GRPCProvider: PlanResourceChange")

	resSchema := p.getResourceSchema(r.TypeName)

	priorMP, err := msgpack.Marshal(r.PriorState, resSchema.Block.ImpliedType())
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(err)
		return resp
	}
	propMP, err := msgpack.Marshal(r.ProposedNewState, resSchema.Block.ImpliedType())
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(err)
		return resp
	}

	protoReq := &proto.PlanResourceChange_Request{
		TypeName:         r.TypeName,
		PriorState:       &proto.DynamicValue{Msgpack: priorMP},
		ProposedNewState: &proto.DynamicValue{Msgpack: propMP},
		PriorPrivate:     r.PriorPrivate,
	}

	protoResp, err := p.client.PlanResourceChange(p.ctx, protoReq)
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(err)
		return resp
	}

	resp.Diagnostics = resp.Diagnostics.Append(convert.ProtoToDiagnostics(protoResp.Diagnostics))

	state := cty.NullVal(resSchema.Block.ImpliedType())
	if protoResp.PlannedState != nil {
		state, err = msgpack.Unmarshal(protoResp.PlannedState.Msgpack, resSchema.Block.ImpliedType())
		if err != nil {
			resp.Diagnostics = resp.Diagnostics.Append(err)
			return resp
		}
	}
	resp.PlannedState = state

	for _, p := range protoResp.RequiresReplace {
		resp.RequiresReplace = append(resp.RequiresReplace, convert.AttributePathToPath(p))
	}

	resp.PlannedPrivate = protoResp.PlannedPrivate

	return resp
}

func (p *GRPCProvider) ApplyResourceChange(r providers.ApplyResourceChangeRequest) (resp providers.ApplyResourceChangeResponse) {
	log.Printf("[TRACE] GRPCProvider: ApplyResourceChange")

	resSchema := p.getResourceSchema(r.TypeName)

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

	protoReq := &proto.ApplyResourceChange_Request{
		TypeName:       r.TypeName,
		PriorState:     &proto.DynamicValue{Msgpack: priorMP},
		PlannedState:   &proto.DynamicValue{Msgpack: plannedMP},
		PlannedPrivate: r.PlannedPrivate,
	}

	protoResp, err := p.client.ApplyResourceChange(p.ctx, protoReq)
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(err)
		return resp
	}

	resp.Private = protoResp.Private

	state := cty.NullVal(resSchema.Block.ImpliedType())
	if protoResp.NewState != nil {
		state, err = msgpack.Unmarshal(protoResp.NewState.Msgpack, resSchema.Block.ImpliedType())
		if err != nil {
			resp.Diagnostics = resp.Diagnostics.Append(err)
			return resp
		}
	}
	resp.NewState = state

	return resp
}

func (p *GRPCProvider) ImportResourceState(r providers.ImportResourceStateRequest) (resp providers.ImportResourceStateResponse) {
	log.Printf("[TRACE] GRPCProvider: ImportResourceState")

	resSchema := p.getResourceSchema(r.TypeName)

	protoReq := &proto.ImportResourceState_Request{
		TypeName: r.TypeName,
		Id:       r.ID,
	}

	protoResp, err := p.client.ImportResourceState(p.ctx, protoReq)
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(err)
		return resp
	}

	for _, imported := range protoResp.ImportedResources {
		resource := providers.ImportedResource{
			TypeName: imported.TypeName,
			Private:  imported.Private,
		}

		state := cty.NullVal(resSchema.Block.ImpliedType())
		if imported.State != nil {
			state, err = msgpack.Unmarshal(imported.State.Msgpack, resSchema.Block.ImpliedType())
			if err != nil {
				resp.Diagnostics = resp.Diagnostics.Append(err)
				return resp
			}
		}
		resource.State = state
		resp.ImportedResources = append(resp.ImportedResources, resource)
	}

	return resp
}

func (p *GRPCProvider) ReadDataSource(r providers.ReadDataSourceRequest) (resp providers.ReadDataSourceResponse) {
	log.Printf("[TRACE] GRPCProvider: ReadDataSource")

	dataSchema := p.getDatasourceSchema(r.TypeName)

	config, err := msgpack.Marshal(r.Config, dataSchema.Block.ImpliedType())
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(err)
		return resp
	}

	protoReq := &proto.ReadDataSource_Request{
		TypeName: r.TypeName,
		Config: &proto.DynamicValue{
			Msgpack: config,
		},
	}

	protoResp, err := p.client.ReadDataSource(p.ctx, protoReq)
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(err)
		return resp
	}
	state := cty.NullVal(dataSchema.Block.ImpliedType())
	if protoResp.State != nil {
		state, err = msgpack.Unmarshal(protoResp.State.Msgpack, dataSchema.Block.ImpliedType())
		if err != nil {
			resp.Diagnostics = resp.Diagnostics.Append(err)
			return resp
		}
	}
	resp.State = state

	return resp
}

// closing the grpc connection is final, and terraform will call it at the end of every phase.
func (p *GRPCProvider) Close() error {
	// check this since it's not automatically inserted during plugin creation
	if p.PluginClient == nil {
		log.Println("[DEBUG] provider has no plugin.Client")
		return nil
	}

	p.PluginClient.Kill()
	return nil
}
