package main

import (
	"github.com/hashicorp/terraform/builtin/providers/ns1"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: ns1.Provider,
	})
}
