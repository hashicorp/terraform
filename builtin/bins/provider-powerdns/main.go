package main

import (
	"github.com/hashicorp/terraform/builtin/providers/powerdns"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: powerdns.Provider,
	})
}
