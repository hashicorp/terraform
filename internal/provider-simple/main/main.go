package main

import (
	"github.com/hashicorp/terraform/internal/grpcwrap"
	simple "github.com/hashicorp/terraform/internal/provider-simple"
	proto "github.com/hashicorp/terraform/internal/tfplugin6"
	plugin "github.com/hashicorp/terraform/plugin6"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		GRPCProviderFunc: func() proto.ProviderServer {
			return grpcwrap.Provider(simple.Provider())
		},
	})
}
