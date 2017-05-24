package main

import (
	"github.com/hashicorp/terraform/builtin/providers/cloudinit"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: cloudinit.Provider,
	})
}
