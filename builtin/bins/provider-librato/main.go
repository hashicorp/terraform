package main

import (
	"github.com/hashicorp/terraform/builtin/providers/librato"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: librato.Provider,
	})
}
