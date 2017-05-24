package main

import (
	"github.com/r3labs/terraform/builtin/provisioners/local-exec"
	"github.com/r3labs/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProvisionerFunc: localexec.Provisioner,
	})
}
