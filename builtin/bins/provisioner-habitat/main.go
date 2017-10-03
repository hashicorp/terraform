package main

import (
	"github.com/hashicorp/terraform/builtin/provisioners/habitat"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProvisionerFunc: habitat.Provisioner,
	})
}
