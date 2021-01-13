package main

import (
	"github.com/hashicorp/terraform/builtin/providers/terraform"
	"github.com/hashicorp/terraform/internal/grpcwrap"
	proto "github.com/hashicorp/terraform/internal/tfplugin6"
	plugin "github.com/hashicorp/terraform/plugin6"
)

func main() {
	// Provide a binary version of the internal terraform provider for testing
	plugin.Serve(&plugin.ServeOpts{
		GRPCProviderFunc: func() proto.ProviderServer {
			return grpcwrap.Provider(terraform.NewProvider())
		},
	})
}
