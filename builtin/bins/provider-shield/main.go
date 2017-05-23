package main

import (
	"github.com/hashicorp/terraform/builtin/providers/shield"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: shield.Provider,
	})
}
