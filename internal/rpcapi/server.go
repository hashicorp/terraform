package rpcapi

import (
	"fmt"

	"go.rpcplugin.org/rpcplugin"
	"google.golang.org/grpc"

	"github.com/hashicorp/terraform/internal/rpcapi/tfcore1"
	"github.com/hashicorp/terraform/internal/terraform"
)

type tfcore1PluginServer struct {
	core *terraform.Context
}

var _ tfcore1.TerraformServer = (*tfcore1PluginServer)(nil)

func newV1PluginServer(core *terraform.Context) tfcore1.TerraformServer {
	return &tfcore1PluginServer{core}
}

type version1 struct {
	getCoreOpts func() *terraform.ContextOpts
}

var _ rpcplugin.ServerVersion = version1{}

func (p version1) RegisterServer(server *grpc.Server) error {
	coreOpts := p.getCoreOpts()
	core, diags := terraform.NewContext(coreOpts)
	if diags.HasErrors() {
		return fmt.Errorf("failed to instantiate Terraform Core: %w", diags.Err())
	}

	tfcore1.RegisterTerraformServer(server, &tfcore1PluginServer{core})
	return nil
}
