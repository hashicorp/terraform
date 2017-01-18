package main

import (
	"github.com/hashicorp/terraform/builtin/providers/profitbricks"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: profitbricks.Provider,
	})
}
