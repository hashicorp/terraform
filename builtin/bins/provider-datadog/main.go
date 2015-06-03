package main

import (
	"github.com/hashicorp/terraform/builtin/providers/datadog"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: datadog.Provider,
	})
}
