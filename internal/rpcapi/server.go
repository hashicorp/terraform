package rpcapi

import (
	"go.rpcplugin.org/rpcplugin"
	"google.golang.org/grpc"

	"github.com/hashicorp/terraform/internal/rpcapi/tfcore1"
)

type tfcorePluginServer struct {
}

var _ tfcore1.TerraformServer = (*tfcorePluginServer)(nil)

type version1 struct {
}

var _ rpcplugin.ServerVersion = version1{}

func (p version1) RegisterServer(server *grpc.Server) error {
	tfcore1.RegisterTerraformServer(server, &tfcorePluginServer{})
	return nil
}
