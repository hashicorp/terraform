package main

import (
	"github.com/hashicorp/terraform/builtin/providers/dyn"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: dyn.Provider,
	})
}
