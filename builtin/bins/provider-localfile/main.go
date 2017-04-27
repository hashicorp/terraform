package main

import (
	"github.com/hashicorp/terraform/builtin/providers/local"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: local.Provider,
	})
}
