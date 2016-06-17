package main

import (
	"github.com/hashicorp/terraform/builtin/providers/ddcloud"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: ddcloud.Provider,
	})
}
