package main

import (
	"github.com/hashicorp/terraform/builtin/providers/ibmcloud"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: ibmcloud.Provider,
	})
}
