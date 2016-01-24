package main

import (
	"github.com/hashicorp/terraform/builtin/providers/postgresql"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: postgresql.Provider,
	})
}
