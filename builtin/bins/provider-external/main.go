package main

import (
	"github.com/hashicorp/terraform/builtin/providers/external"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: external.Provider,
	})
}
