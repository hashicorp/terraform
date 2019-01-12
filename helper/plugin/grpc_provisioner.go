package plugin

import (
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	proto "github.com/hashicorp/terraform/internal/tfplugin5"
	"github.com/hashicorp/terraform/plugin/convert"
	"github.com/hashicorp/terraform/terraform"
	"github.com/zclconf/go-cty/cty"
	ctyconvert "github.com/zclconf/go-cty/cty/convert"
	"github.com/zclconf/go-cty/cty/msgpack"
	context "golang.org/x/net/context"
)

// NewGRPCProvisionerServerShim wraps a terraform.ResourceProvisioner in a
// proto.ProvisionerServer implementation. If the provided provisioner is not a
// *schema.Provisioner, this will return nil,
func NewGRPCProvisionerServerShim(p terraform.ResourceProvisioner) *GRPCProvisionerServer {
	sp, ok := p.(*schema.Provisioner)
	if !ok {
		return nil
	}
	return &GRPCProvisionerServer{
		provisioner: sp,
	}
}

type GRPCProvisionerServer struct {
	provisioner *schema.Provisioner
}

func (s *GRPCProvisionerServer) GetSchema(_ context.Context, req *proto.GetProvisionerSchema_Request) (*proto.GetProvisionerSchema_Response, error) {
	resp := &proto.GetProvisionerSchema_Response{}

	resp.Provisioner = &proto.Schema{
		Block: convert.ConfigSchemaToProto(schema.InternalMap(s.provisioner.Schema).CoreConfigSchema()),
	}

	return resp, nil
}

func (s *GRPCProvisionerServer) ValidateProvisionerConfig(_ context.Context, req *proto.ValidateProvisionerConfig_Request) (*proto.ValidateProvisionerConfig_Response, error) {
	resp := &proto.ValidateProvisionerConfig_Response{}

	cfgSchema := schema.InternalMap(s.provisioner.Schema).CoreConfigSchema()

	configVal, err := msgpack.Unmarshal(req.Config.Msgpack, cfgSchema.ImpliedType())
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	config := terraform.NewResourceConfigShimmed(configVal, cfgSchema)

	warns, errs := s.provisioner.Validate(config)
	resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, convert.WarnsAndErrsToProto(warns, errs))

	return resp, nil
}

// stringMapFromValue converts a cty.Value to a map[stirng]string.
// This will panic if the val is not a cty.Map(cty.String).
func stringMapFromValue(val cty.Value) map[string]string {
	m := map[string]string{}
	if val.IsNull() || !val.IsKnown() {
		return m
	}

	for it := val.ElementIterator(); it.Next(); {
		ak, av := it.Element()
		name := ak.AsString()

		if !av.IsKnown() || av.IsNull() {
			continue
		}

		av, _ = ctyconvert.Convert(av, cty.String)
		m[name] = av.AsString()
	}

	return m
}

// uiOutput implements the terraform.UIOutput interface to adapt the grpc
// stream to the legacy Provisioner.Apply method.
type uiOutput struct {
	srv proto.Provisioner_ProvisionResourceServer
}

func (o uiOutput) Output(s string) {
	err := o.srv.Send(&proto.ProvisionResource_Response{
		Output: s,
	})
	if err != nil {
		log.Printf("[ERROR] %s", err)
	}
}

func (s *GRPCProvisionerServer) ProvisionResource(req *proto.ProvisionResource_Request, srv proto.Provisioner_ProvisionResourceServer) error {
	// We send back a diagnostics over the stream if there was a
	// provisioner-side problem.
	srvResp := &proto.ProvisionResource_Response{}

	cfgSchema := schema.InternalMap(s.provisioner.Schema).CoreConfigSchema()
	cfgVal, err := msgpack.Unmarshal(req.Config.Msgpack, cfgSchema.ImpliedType())
	if err != nil {
		srvResp.Diagnostics = convert.AppendProtoDiag(srvResp.Diagnostics, err)
		srv.Send(srvResp)
		return nil
	}
	resourceConfig := terraform.NewResourceConfigShimmed(cfgVal, cfgSchema)

	connVal, err := msgpack.Unmarshal(req.Connection.Msgpack, cty.Map(cty.String))
	if err != nil {
		srvResp.Diagnostics = convert.AppendProtoDiag(srvResp.Diagnostics, err)
		srv.Send(srvResp)
		return nil
	}

	conn := stringMapFromValue(connVal)

	instanceState := &terraform.InstanceState{
		Ephemeral: terraform.EphemeralState{
			ConnInfo: conn,
		},
		Meta: make(map[string]interface{}),
	}

	err = s.provisioner.Apply(uiOutput{srv}, instanceState, resourceConfig)
	if err != nil {
		srvResp.Diagnostics = convert.AppendProtoDiag(srvResp.Diagnostics, err)
		srv.Send(srvResp)
	}
	return nil
}

func (s *GRPCProvisionerServer) Stop(_ context.Context, req *proto.Stop_Request) (*proto.Stop_Response, error) {
	resp := &proto.Stop_Response{}

	err := s.provisioner.Stop()
	if err != nil {
		resp.Error = err.Error()
	}

	return resp, nil
}
