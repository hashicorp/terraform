package stressprovider

import (
	"context"
	"fmt"
	"net/rpc"

	"github.com/hashicorp/go-plugin"
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"
	"github.com/zclconf/go-cty/cty/msgpack"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/hashicorp/terraform/internal/tfplugin5"
	pluginConvert "github.com/hashicorp/terraform/plugin/convert"
	"github.com/hashicorp/terraform/providers"
)

// While we're doing normal stress-testing we just run the stressprovider
// in-process and access its API via normal function calls, but we also have
// an RPC implementation of it which is used by the "stresstest terraform"
// command so that the provider can be made available to a normal Terraform CLI
// process while debugging a test failure.

type Plugin struct {
	base *Provider
}

var _ plugin.GRPCPlugin = (*Plugin)(nil)
var _ plugin.Plugin = (*Plugin)(nil)

func (p *Provider) Plugin() *Plugin {
	return &Plugin{p}
}

func (p *Plugin) GRPCServer(broker *plugin.GRPCBroker, server *grpc.Server) error {
	inst := p.base.NewInstance()
	tfplugin5.RegisterProviderServer(server, inst.Server())
	return nil
}

func (p *Plugin) GRPCClient(context.Context, *plugin.GRPCBroker, *grpc.ClientConn) (interface{}, error) {
	return nil, fmt.Errorf("this is only a server")
}

func (p *Plugin) Server(*plugin.MuxBroker) (interface{}, error) {
	return nil, fmt.Errorf("go-plugin net/rpc is obsolete")
}

func (p *Plugin) Client(*plugin.MuxBroker, *rpc.Client) (interface{}, error) {
	return nil, fmt.Errorf("go-plugin net/rpc is obsolete")
}

type Server struct {
	provider *Provider
}

func (p *Provider) Server() *Server {
	return &Server{p}
}

var _ tfplugin5.ProviderServer = (*Server)(nil)

func (s *Server) GetSchema(ctx context.Context, req *tfplugin5.GetProviderSchema_Request) (*tfplugin5.GetProviderSchema_Response, error) {
	resp := s.provider.GetSchema()
	ret := &tfplugin5.GetProviderSchema_Response{
		Provider: &tfplugin5.Schema{
			Block: pluginConvert.ConfigSchemaToProto(resp.Provider.Block),
		},
	}
	ret.Diagnostics = pluginConvert.AppendProtoDiag(ret.Diagnostics, resp.Diagnostics)
	ret.ResourceSchemas = make(map[string]*tfplugin5.Schema)
	for typeName, schema := range resp.ResourceTypes {
		ret.ResourceSchemas[typeName] = &tfplugin5.Schema{
			Block: pluginConvert.ConfigSchemaToProto(schema.Block),
		}
	}
	ret.DataSourceSchemas = make(map[string]*tfplugin5.Schema)
	for typeName, schema := range resp.DataSources {
		ret.DataSourceSchemas[typeName] = &tfplugin5.Schema{
			Block: pluginConvert.ConfigSchemaToProto(schema.Block),
		}
	}
	return ret, nil
}

func (s *Server) PrepareProviderConfig(ctx context.Context, req *tfplugin5.PrepareProviderConfig_Request) (*tfplugin5.PrepareProviderConfig_Response, error) {
	ret := &tfplugin5.PrepareProviderConfig_Response{}
	rawVal := req.Config
	ty := providerConfigSchema.Block.ImpliedType()
	v, err := decodeDynamicValue(rawVal, ty)
	if err != nil {
		ret.Diagnostics = pluginConvert.AppendProtoDiag(ret.Diagnostics, err)
		return ret, nil
	}
	resp := s.provider.PrepareProviderConfig(providers.PrepareProviderConfigRequest{
		Config: v,
	})
	ret.Diagnostics = pluginConvert.AppendProtoDiag(ret.Diagnostics, resp.Diagnostics)
	rawVal, err = encodeDynamicValue(resp.PreparedConfig, ty)
	if err != nil {
		ret.Diagnostics = pluginConvert.AppendProtoDiag(ret.Diagnostics, err)
		return ret, nil
	}
	ret.PreparedConfig = rawVal
	return ret, nil
}

func (s *Server) ValidateResourceTypeConfig(ctx context.Context, req *tfplugin5.ValidateResourceTypeConfig_Request) (*tfplugin5.ValidateResourceTypeConfig_Response, error) {
	// The underlying provider's ValidateResourceTypeConfig method doesn't
	// do anything, so we won't do anything here either to avoid wasting
	// time writing translation code that will never run.
	return &tfplugin5.ValidateResourceTypeConfig_Response{}, nil
}

func (s *Server) ValidateDataSourceConfig(ctx context.Context, req *tfplugin5.ValidateDataSourceConfig_Request) (*tfplugin5.ValidateDataSourceConfig_Response, error) {
	// The underlying provider's ValidateDataSourceConfig method doesn't
	// do anything, so we won't do anything here either to avoid wasting
	// time writing translation code that will never run.
	return &tfplugin5.ValidateDataSourceConfig_Response{}, nil
}

func (s *Server) UpgradeResourceState(ctx context.Context, req *tfplugin5.UpgradeResourceState_Request) (*tfplugin5.UpgradeResourceState_Response, error) {
	ty := ManagedResourceTypeSchema.Block.ImpliedType()
	ret := &tfplugin5.UpgradeResourceState_Response{}
	resp := s.provider.UpgradeResourceState(providers.UpgradeResourceStateRequest{
		TypeName:        req.TypeName,
		Version:         req.Version,
		RawStateJSON:    req.RawState.Json,
		RawStateFlatmap: req.RawState.Flatmap,
	})
	ret.Diagnostics = pluginConvert.AppendProtoDiag(ret.Diagnostics, resp.Diagnostics)
	rawVal, err := encodeDynamicValue(resp.UpgradedState, ty)
	if err != nil {
		ret.Diagnostics = pluginConvert.AppendProtoDiag(ret.Diagnostics, err)
		return ret, nil
	}
	ret.UpgradedState = rawVal
	return ret, nil
}

func (s *Server) Configure(ctx context.Context, req *tfplugin5.Configure_Request) (*tfplugin5.Configure_Response, error) {
	ret := &tfplugin5.Configure_Response{}
	rawVal := req.Config
	ty := providerConfigSchema.Block.ImpliedType()
	v, err := decodeDynamicValue(rawVal, ty)
	if err != nil {
		ret.Diagnostics = pluginConvert.AppendProtoDiag(ret.Diagnostics, err)
		return ret, nil
	}
	resp := s.provider.Configure(providers.ConfigureRequest{
		Config: v,
	})
	ret.Diagnostics = pluginConvert.AppendProtoDiag(ret.Diagnostics, resp.Diagnostics)
	return ret, nil
}

func (s *Server) ReadResource(ctx context.Context, req *tfplugin5.ReadResource_Request) (*tfplugin5.ReadResource_Response, error) {
	ret := &tfplugin5.ReadResource_Response{}
	rawVal := req.CurrentState
	ty := ManagedResourceTypeSchema.Block.ImpliedType()
	v, err := decodeDynamicValue(rawVal, ty)
	if err != nil {
		ret.Diagnostics = pluginConvert.AppendProtoDiag(ret.Diagnostics, err)
		return ret, nil
	}
	resp := s.provider.ReadResource(providers.ReadResourceRequest{
		TypeName:   req.TypeName,
		PriorState: v,
	})
	ret.Diagnostics = pluginConvert.AppendProtoDiag(ret.Diagnostics, resp.Diagnostics)
	rawVal, err = encodeDynamicValue(resp.NewState, ty)
	if err != nil {
		ret.Diagnostics = pluginConvert.AppendProtoDiag(ret.Diagnostics, err)
		return ret, nil
	}
	ret.NewState = rawVal
	return ret, nil
}

func (s *Server) PlanResourceChange(ctx context.Context, req *tfplugin5.PlanResourceChange_Request) (*tfplugin5.PlanResourceChange_Response, error) {
	ret := &tfplugin5.PlanResourceChange_Response{}
	ty := ManagedResourceTypeSchema.Block.ImpliedType()
	rawVal := req.Config
	configVal, err := decodeDynamicValue(rawVal, ty)
	if err != nil {
		ret.Diagnostics = pluginConvert.AppendProtoDiag(ret.Diagnostics, err)
		return ret, nil
	}
	rawVal = req.PriorState
	priorStateVal, err := decodeDynamicValue(rawVal, ty)
	if err != nil {
		ret.Diagnostics = pluginConvert.AppendProtoDiag(ret.Diagnostics, err)
		return ret, nil
	}
	rawVal = req.ProposedNewState
	proposedNewStateVal, err := decodeDynamicValue(rawVal, ty)
	if err != nil {
		ret.Diagnostics = pluginConvert.AppendProtoDiag(ret.Diagnostics, err)
		return ret, nil
	}
	resp := s.provider.PlanResourceChange(providers.PlanResourceChangeRequest{
		TypeName:         req.TypeName,
		ProposedNewState: proposedNewStateVal,
		Config:           configVal,
		PriorState:       priorStateVal,
	})
	ret.Diagnostics = pluginConvert.AppendProtoDiag(ret.Diagnostics, resp.Diagnostics)
	rawVal, err = encodeDynamicValue(resp.PlannedState, ty)
	if err != nil {
		ret.Diagnostics = pluginConvert.AppendProtoDiag(ret.Diagnostics, err)
		return ret, nil
	}
	ret.PlannedState = rawVal
	return ret, nil
}

func (s *Server) ApplyResourceChange(ctx context.Context, req *tfplugin5.ApplyResourceChange_Request) (*tfplugin5.ApplyResourceChange_Response, error) {
	ret := &tfplugin5.ApplyResourceChange_Response{}
	ty := ManagedResourceTypeSchema.Block.ImpliedType()
	rawVal := req.Config
	configVal, err := decodeDynamicValue(rawVal, ty)
	if err != nil {
		ret.Diagnostics = pluginConvert.AppendProtoDiag(ret.Diagnostics, err)
		return ret, nil
	}
	rawVal = req.PriorState
	priorStateVal, err := decodeDynamicValue(rawVal, ty)
	if err != nil {
		ret.Diagnostics = pluginConvert.AppendProtoDiag(ret.Diagnostics, err)
		return ret, nil
	}
	rawVal = req.PlannedState
	plannedStateVal, err := decodeDynamicValue(rawVal, ty)
	if err != nil {
		ret.Diagnostics = pluginConvert.AppendProtoDiag(ret.Diagnostics, err)
		return ret, nil
	}
	resp := s.provider.ApplyResourceChange(providers.ApplyResourceChangeRequest{
		TypeName:     req.TypeName,
		PlannedState: plannedStateVal,
		Config:       configVal,
		PriorState:   priorStateVal,
	})
	ret.Diagnostics = pluginConvert.AppendProtoDiag(ret.Diagnostics, resp.Diagnostics)
	rawVal, err = encodeDynamicValue(resp.NewState, ty)
	if err != nil {
		ret.Diagnostics = pluginConvert.AppendProtoDiag(ret.Diagnostics, err)
		return ret, nil
	}
	ret.NewState = rawVal
	return ret, nil
}

func (s *Server) ImportResourceState(ctx context.Context, req *tfplugin5.ImportResourceState_Request) (*tfplugin5.ImportResourceState_Response, error) {
	return nil, grpc.Errorf(codes.Unimplemented, "not implemented")
}

func (s *Server) ReadDataSource(ctx context.Context, req *tfplugin5.ReadDataSource_Request) (*tfplugin5.ReadDataSource_Response, error) {
	ret := &tfplugin5.ReadDataSource_Response{}
	rawVal := req.Config
	ty := DataResourceTypeSchema.Block.ImpliedType()
	v, err := decodeDynamicValue(rawVal, ty)
	if err != nil {
		ret.Diagnostics = pluginConvert.AppendProtoDiag(ret.Diagnostics, err)
		return ret, nil
	}
	resp := s.provider.ReadDataSource(providers.ReadDataSourceRequest{
		TypeName: req.TypeName,
		Config:   v,
	})
	ret.Diagnostics = pluginConvert.AppendProtoDiag(ret.Diagnostics, resp.Diagnostics)
	rawVal, err = encodeDynamicValue(resp.State, ty)
	if err != nil {
		ret.Diagnostics = pluginConvert.AppendProtoDiag(ret.Diagnostics, err)
		return ret, nil
	}
	ret.State = rawVal
	return ret, nil
}

func (s *Server) Stop(ctx context.Context, req *tfplugin5.Stop_Request) (*tfplugin5.Stop_Response, error) {
	// This provider isn't stoppable, because it's doing all of its work
	// locally in memory anyway.
	return nil, nil
}

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

func encodeDynamicValue(v cty.Value, ty cty.Type) (*tfplugin5.DynamicValue, error) {
	raw, err := msgpack.Marshal(v, ty)
	if err != nil {
		return nil, err
	}
	return &tfplugin5.DynamicValue{
		Msgpack: raw,
	}, nil
}
