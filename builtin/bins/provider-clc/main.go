package main

import (
	"github.com/hashicorp/terraform/builtin/providers/clc"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: clc.Provider,
	})
}
