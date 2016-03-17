package main

import (
	"github.com/hashicorp/terraform/builtin/providers/github"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: github.Provider,
	})
}
