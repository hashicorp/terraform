package main

import (
	"github.com/r3labs/terraform/builtin/providers/rundeck"
	"github.com/r3labs/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: rundeck.Provider,
	})
}
