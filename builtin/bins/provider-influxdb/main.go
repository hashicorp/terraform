package main

import (
	"github.com/hashicorp/terraform/builtin/providers/influxdb"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: influxdb.Provider,
	})
}
