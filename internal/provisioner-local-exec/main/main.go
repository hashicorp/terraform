// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	localexec "github.com/hashicorp/mnptu/internal/builtin/provisioners/local-exec"
	"github.com/hashicorp/mnptu/internal/grpcwrap"
	"github.com/hashicorp/mnptu/internal/plugin"
	"github.com/hashicorp/mnptu/internal/tfplugin5"
)

func main() {
	// Provide a binary version of the internal mnptu provider for testing
	plugin.Serve(&plugin.ServeOpts{
		GRPCProvisionerFunc: func() tfplugin5.ProvisionerServer {
			return grpcwrap.Provisioner(localexec.New())
		},
	})
}
