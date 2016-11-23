package main

import (
	"github.com/hashicorp/terraform/builtin/providers/cloudfoundry"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: cloudfoundry.Provider,
	})
}
