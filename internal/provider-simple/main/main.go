// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"github.com/hashicorp/mnptu/internal/grpcwrap"
	"github.com/hashicorp/mnptu/internal/plugin"
	simple "github.com/hashicorp/mnptu/internal/provider-simple"
	"github.com/hashicorp/mnptu/internal/tfplugin5"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		GRPCProviderFunc: func() tfplugin5.ProviderServer {
			return grpcwrap.Provider(simple.Provider())
		},
	})
}
