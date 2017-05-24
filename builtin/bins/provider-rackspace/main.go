package main

import (
	"github.com/hashicorp/terraform/builtin/providers/rackspace"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: rackspace.Provider,
	})
}
