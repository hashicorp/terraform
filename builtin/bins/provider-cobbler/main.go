package main

import (
	"github.com/hashicorp/terraform/builtin/providers/cobbler"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: cobbler.Provider,
	})
}
