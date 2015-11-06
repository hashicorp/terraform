package main

import (
	"github.com/hashicorp/terraform/builtin/providers/vsphere"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: vsphere.Provider,
	})
}
