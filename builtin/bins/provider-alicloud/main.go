package main

import (
	"github.com/hashicorp/terraform/builtin/providers/alicloud"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: alicloud.Provider,
	})
}