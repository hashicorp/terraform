package main

import (
	"github.com/hashicorp/terraform/builtin/provisioners/ansible"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProvisionerFunc: ansible.Provisioner,
	})
}
