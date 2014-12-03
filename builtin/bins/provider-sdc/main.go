package main

import (
	"github.com/hashicorp/terraform/builtin/providers/sdc"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: sdc.Provider,
	})
}
