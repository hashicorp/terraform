package main

import (
	"github.com/hashicorp/terraform/builtin/providers/occi"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: occi.Provider,
	})
}
