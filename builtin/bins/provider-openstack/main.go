package main

import (
	"github.com/r3labs/terraform/builtin/providers/openstack"
	"github.com/r3labs/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: openstack.Provider,
	})
}
