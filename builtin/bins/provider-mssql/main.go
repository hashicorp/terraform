package main

import (
	"github.com/hashicorp/terraform/builtin/providers/mssql"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: mssql.Provider,
	})
}
