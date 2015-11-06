package main

import (
	"github.com/hashicorp/terraform/builtin/providers/rundeck"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: rundeck.Provider,
	})
}
