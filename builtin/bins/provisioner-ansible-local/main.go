package main

import (
	"github.com/hashicorp/terraform/builtin/provisioners/ansible-local"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProvisionerFunc: ansible_local.Provisioner,
	})
}
