package main

import (
	"github.com/hashicorp/terraform/plugin"
	"github.com/hashicorp/terraform/builtin/providers/opsgenie"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: opsgenie.Provider,
	})
}
