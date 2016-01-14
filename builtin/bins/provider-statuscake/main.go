package main

import (
	"github.com/hashicorp/terraform/builtin/providers/statuscake"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: statuscake.Provider,
	})
}
