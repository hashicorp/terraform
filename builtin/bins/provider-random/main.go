package main

import (
	"github.com/hashicorp/terraform/builtin/providers/random"
	"github.com/hashicorp/terraform/plugin"
	"github.com/hashicorp/terraform/terraform"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: func() terraform.ResourceProvider {
			return random.Provider()
		},
	})
}
