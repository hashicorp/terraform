package main

import (
	"github.com/r3labs/terraform/builtin/providers/dme"
	"github.com/r3labs/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: dme.Provider,
	})
}
