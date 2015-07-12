package main

import (
	"github.com/hashicorp/terraform/builtin/providers/template"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: template.Provider,
	})
}
