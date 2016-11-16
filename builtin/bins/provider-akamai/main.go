package main

import (
	"github.com/hashicorp/terraform/builtin/providers/akamai"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: akamai.Provider,
	})
}
