package main

import (
	"github.com/hashicorp/terraform/builtin/providers/softlayer"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: softlayer.Provider,
	})
}
