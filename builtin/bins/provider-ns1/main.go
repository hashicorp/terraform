package main

import (
	"github.com/r3labs/terraform/builtin/providers/ns1"
	"github.com/r3labs/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: ns1.Provider,
	})
}
