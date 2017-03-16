package main

import (
	"github.com/hashicorp/terraform/builtin/provisioners/file"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProvisionerFunc: file.Provisioner,
	})
}
