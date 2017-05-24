package main

import (
	"github.com/r3labs/terraform/builtin/providers/rancher"
	"github.com/r3labs/terraform/plugin"
	"github.com/r3labs/terraform/terraform"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: func() terraform.ResourceProvider {
			return rancher.Provider()
		},
	})
}
