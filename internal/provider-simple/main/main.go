package main

import (
	"github.com/hashicorp/terraform/internal/grpcwrap"
	"github.com/hashicorp/terraform/internal/plugin"
	simple "github.com/hashicorp/terraform/internal/provider-simple"
	"github.com/hashicorp/terraform/internal/tfplugin5"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		GRPCProviderFunc: func() tfplugin5.ProviderServer {
			return grpcwrap.Provider(simple.Provider())
		},
	})
}
