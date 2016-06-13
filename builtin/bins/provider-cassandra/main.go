package main

import (
	"github.com/hashicorp/terraform/builtin/providers/cassandra"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: cassandra.Provider,
	})
}
