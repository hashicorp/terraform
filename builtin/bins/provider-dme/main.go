package main

import (
	"github.com/hashicorp/terraform/builtin/providers/dme"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: dme.Provider,
	})
}
