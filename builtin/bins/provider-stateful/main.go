package main

import (
	"github.com/hashicorp/terraform/builtin/providers/stateful"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: stateful.Provider,
	})
}
