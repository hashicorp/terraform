package main

import (
	"github.com/hashicorp/terraform/builtin/providers/chef"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: chef.Provider,
	})
}
