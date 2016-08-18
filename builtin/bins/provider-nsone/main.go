package main

import (
	"github.com/hashicorp/terraform/builtin/providers/nsone"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: nsone.Provider,
	})
}
