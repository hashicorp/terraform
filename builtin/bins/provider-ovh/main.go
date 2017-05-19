package main

import (
	"github.com/hashicorp/terraform/builtin/providers/ovh"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: ovh.Provider,
	})
}
