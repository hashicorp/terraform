package main

import (
	"github.com/hashicorp/terraform/builtin/providers/bitbucket"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: bitbucket.Provider,
	})
}
