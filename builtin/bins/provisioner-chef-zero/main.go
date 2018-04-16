package main

import (
	"github.com/hashicorp/terraform/builtin/provisioners/chef-zero"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProvisionerFunc: chef.Provisioner,
	})
}
