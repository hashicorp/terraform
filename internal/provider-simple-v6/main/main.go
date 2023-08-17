// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"github.com/hashicorp/mnptu/internal/grpcwrap"
	plugin "github.com/hashicorp/mnptu/internal/plugin6"
	simple "github.com/hashicorp/mnptu/internal/provider-simple-v6"
	"github.com/hashicorp/mnptu/internal/tfplugin6"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		GRPCProviderFunc: func() tfplugin6.ProviderServer {
			return grpcwrap.Provider6(simple.Provider())
		},
	})
}
