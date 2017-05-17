package main

import (
	"github.com/hashicorp/terraform/builtin/providers/pagerduty"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: pagerduty.Provider,
	})
}
