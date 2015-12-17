package main

import (
	"github.com/hashicorp/terraform/builtin/providers/mysql"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: mysql.Provider,
	})
}
