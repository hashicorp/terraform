package main

import (
	"github.com/hashicorp/terraform/builtin/provisioners/local-exec"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(new(localexec.ResourceProvisioner))
}
