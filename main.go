package main

import (
	"github.com/hashicorp/terraform/plugin"
	"github.com/terraform-providers/terraform-provider-softlayer/softlayer"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: softlayer.Provider})
}
