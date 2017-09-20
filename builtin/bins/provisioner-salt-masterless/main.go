package main

import (
	"github.com/hashicorp/terraform/builtin/provisioners/salt-masterless"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProvisionerFunc: saltmasterless.Provisioner,
	})
}
