// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"github.com/hashicorp/mnptu/internal/builtin/providers/mnptu"
	"github.com/hashicorp/mnptu/internal/grpcwrap"
	"github.com/hashicorp/mnptu/internal/plugin"
	"github.com/hashicorp/mnptu/internal/tfplugin5"
)

func main() {
	// Provide a binary version of the internal mnptu provider for testing
	plugin.Serve(&plugin.ServeOpts{
		GRPCProviderFunc: func() tfplugin5.ProviderServer {
			return grpcwrap.Provider(mnptu.NewProvider())
		},
	})
}
