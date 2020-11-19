package main

import (
	"github.com/hashicorp/terraform/builtin/providers/terraform"
	"github.com/hashicorp/terraform/internal/grpcwrap"
	"github.com/hashicorp/terraform/internal/tfplugin5"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	// Provide a binary version of the internal terraform provider for testing
	plugin.Serve(&plugin.ServeOpts{
		GRPCProviderFunc: func() tfplugin5.ProviderServer {
			return grpcwrap.New(terraform.NewProvider())
		},
	})
}
