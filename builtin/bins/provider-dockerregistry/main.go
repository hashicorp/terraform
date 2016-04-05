package main

import (
	"github.com/hashicorp/terraform/builtin/providers/dockerregistry"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: dockerregistry.Provider,
	})
}
