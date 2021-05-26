package main

import (
	localexec "github.com/hashicorp/terraform/internal/builtin/provisioners/local-exec"
	"github.com/hashicorp/terraform/internal/grpcwrap"
	"github.com/hashicorp/terraform/internal/plugin"
	"github.com/hashicorp/terraform/internal/tfplugin5"
)

func main() {
	// Provide a binary version of the internal terraform provider for testing
	plugin.Serve(&plugin.ServeOpts{
		GRPCProvisionerFunc: func() tfplugin5.ProvisionerServer {
			return grpcwrap.Provisioner(localexec.New())
		},
	})
}
