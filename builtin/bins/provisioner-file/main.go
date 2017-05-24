package main

import (
	"github.com/r3labs/terraform/builtin/provisioners/file"
	"github.com/r3labs/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProvisionerFunc: file.Provisioner,
	})
}
