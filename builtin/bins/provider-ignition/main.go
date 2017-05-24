package main

import (
	"github.com/hashicorp/terraform/builtin/providers/ignition"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: ignition.Provider,
	})
}
