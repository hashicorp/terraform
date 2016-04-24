package main

import (
	"github.com/hashicorp/terraform/builtin/providers/fastly"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: fastly.Provider,
	})
}
