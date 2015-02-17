package main

import (
	"github.com/hashicorp/terraform/builtin/providers/docker"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: docker.Provider,
	})
}
