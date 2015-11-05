package main

import (
	"github.com/hashicorp/terraform/builtin/providers/vcd"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: vcd.Provider,
	})
}
