package main

import (
	"github.com/bobtfish/terraform-provider-nsone/nsone"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: nsone.Provider,
	})
}
