package main

import (
	"github.com/r3labs/terraform/builtin/providers/ignition"
	"github.com/r3labs/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: ignition.Provider,
	})
}
