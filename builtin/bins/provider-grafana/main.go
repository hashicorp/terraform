package main

import (
	"github.com/hashicorp/terraform/builtin/providers/grafana"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: grafana.Provider,
	})
}
