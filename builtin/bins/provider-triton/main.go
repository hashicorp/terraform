package main

import (
	"github.com/hashicorp/terraform/builtin/providers/triton"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: triton.Provider,
	})
}
