package main

import (
	"github.com/r3labs/terraform/builtin/providers/consul"
	"github.com/r3labs/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: consul.Provider,
	})
}
