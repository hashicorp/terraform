package main

import (
	"github.com/hashicorp/terraform/builtin/provisioners/remote-exec"
	"github.com/hashicorp/terraform/plugin"
	"github.com/hashicorp/terraform/terraform"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProvisionerFunc: func() terraform.ResourceProvisioner {
			return new(remoteexec.ResourceProvisioner)
		},
	})
}
