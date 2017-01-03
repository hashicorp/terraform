package main

import (
	"github.com/hashicorp/terraform/builtin/providers/archive"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: archive.Provider,
	})
}
