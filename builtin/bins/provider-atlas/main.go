package main

import (
	"github.com/hashicorp/terraform/builtin/providers/atlas"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: atlas.Provider,
	})
}
