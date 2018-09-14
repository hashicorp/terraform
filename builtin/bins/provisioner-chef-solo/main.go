package main

import (
	"github.com/hashicorp/terraform/builtin/provisioners/chef-solo"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProvisionerFunc: chefsolo.Provisioner,
	})
}
