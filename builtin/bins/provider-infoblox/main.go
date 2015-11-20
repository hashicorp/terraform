package main

import (
	"github.com/hashicorp/terraform/builtin/providers/infoblox"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: infoblox.Provider,
	})
}
