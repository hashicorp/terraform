package main

import (
	"github.com/hashicorp/terraform/plugin"
	"github.com/segmentio/terraform/builtin/providers/datadog"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: datadog.Provider,
	})
}
