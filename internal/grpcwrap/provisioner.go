package grpcwrap

import (
	"context"
	"log"
	"strings"
	"unicode/utf8"

	"github.com/hashicorp/terraform/internal/communicator/shared"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/plugin/convert"
	"github.com/hashicorp/terraform/internal/provisioners"
	"github.com/hashicorp/terraform/internal/tfplugin5"
)

// New wraps a provisioners.Interface to implement a grpc ProviderServer.
// This is useful for creating a test binary out of an internal provider
// implementation.
func Provisioner(p provisioners.Interface) tfplugin5.ProvisionerServer {
	return &provisioner{
		provisioner: p,
		schema:      p.GetSchema().Provisioner,
	}
}

type provisioner struct {
	provisioner provisioners.Interface
	schema      *configschema.Block
}

func (p *provisioner) GetSchema(_ context.Context, req *tfplugin5.GetProvisionerSchema_Request) (*tfplugin5.GetProvisionerSchema_Response, error) {
	resp := &tfplugin5.GetProvisionerSchema_Response{}

	resp.Provisioner = &tfplugin5.Schema{
		Block: &tfplugin5.Schema_Block{},
	}

	if p.schema != nil {
		resp.Provisioner.Block = convert.ConfigSchemaToProto(p.schema)
	}

	return resp, nil
}

func (p *provisioner) ValidateProvisionerConfig(_ context.Context, req *tfplugin5.ValidateProvisionerConfig_Request) (*tfplugin5.ValidateProvisionerConfig_Response, error) {
	resp := &tfplugin5.ValidateProvisionerConfig_Response{}
	ty := p.schema.ImpliedType()

	configVal, err := decodeDynamicValue(req.Config, ty)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	validateResp := p.provisioner.ValidateProvisionerConfig(provisioners.ValidateProvisionerConfigRequest{
		Config: configVal,
	})

	resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, validateResp.Diagnostics)
	return resp, nil
}

func (p *provisioner) ProvisionResource(req *tfplugin5.ProvisionResource_Request, srv tfplugin5.Provisioner_ProvisionResourceServer) error {
	// We send back a diagnostics over the stream if there was a
	// provisioner-side problem.
	srvResp := &tfplugin5.ProvisionResource_Response{}

	ty := p.schema.ImpliedType()
	configVal, err := decodeDynamicValue(req.Config, ty)
	if err != nil {
		srvResp.Diagnostics = convert.AppendProtoDiag(srvResp.Diagnostics, err)
		srv.Send(srvResp)
		return nil
	}

	connVal, err := decodeDynamicValue(req.Connection, shared.ConnectionBlockSupersetSchema.ImpliedType())
	if err != nil {
		srvResp.Diagnostics = convert.AppendProtoDiag(srvResp.Diagnostics, err)
		srv.Send(srvResp)
		return nil
	}

	resp := p.provisioner.ProvisionResource(provisioners.ProvisionResourceRequest{
		Config:     configVal,
		Connection: connVal,
		UIOutput:   uiOutput{srv},
	})

	srvResp.Diagnostics = convert.AppendProtoDiag(srvResp.Diagnostics, resp.Diagnostics)
	srv.Send(srvResp)
	return nil
}

func (p *provisioner) Stop(context.Context, *tfplugin5.Stop_Request) (*tfplugin5.Stop_Response, error) {
	resp := &tfplugin5.Stop_Response{}
	err := p.provisioner.Stop()
	if err != nil {
		resp.Error = err.Error()
	}
	return resp, nil
}

// uiOutput implements the terraform.UIOutput interface to adapt the grpc
// stream to the legacy Provisioner.Apply method.
type uiOutput struct {
	srv tfplugin5.Provisioner_ProvisionResourceServer
}

func (o uiOutput) Output(s string) {
	err := o.srv.Send(&tfplugin5.ProvisionResource_Response{
		Output: strings.ToValidUTF8(s, string(utf8.RuneError)),
	})
	if err != nil {
		log.Printf("[ERROR] %s", err)
	}
}
