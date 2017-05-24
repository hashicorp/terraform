package main

import (
	"github.com/r3labs/terraform/builtin/providers/grafana"
	"github.com/r3labs/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: grafana.Provider,
	})
}
