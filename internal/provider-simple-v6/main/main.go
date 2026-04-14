// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"github.com/hashicorp/terraform/internal/grpcwrap"
	plugin "github.com/hashicorp/terraform/internal/plugin6"
	simple "github.com/hashicorp/terraform/internal/provider-simple-v6"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfplugin6"
)

// parentStateDir can be overriden via ldflags
var parentStateDir = ""

func main() {
	var p providers.Interface
	if parentStateDir != "" {
		p = simple.ProviderWithParentStatePath(parentStateDir)
	} else {
		p = simple.Provider()
	}
	plugin.Serve(&plugin.ServeOpts{
		GRPCProviderFunc: func() tfplugin6.ProviderServer {
			return grpcwrap.Provider6(p)
		},
	})
}
