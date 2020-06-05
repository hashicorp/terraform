package main

import (
	"github.com/hashicorp/terraform/builtin/provisioners/puppet"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProvisionerFunc: puppet.Provisioner,
	})
}
