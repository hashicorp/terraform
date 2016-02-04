package main

import (
	"github.com/hashicorp/terraform/builtin/providers/spotinst"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: spotinst.Provider,
	})
}
