package main

import (
	"github.com/hashicorp/terraform/builtin/providers/tutum"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: tutum.Provider,
	})
}
