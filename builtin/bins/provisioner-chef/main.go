package main

import (
	"github.com/hashicorp/terraform/builtin/provisioners/chef"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProvisionerFunc: chef.Provisioner,
	})
}
