package main

import (
	"github.com/hashicorp/terraform/builtin/providers/logentries"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: logentries.Provider,
	})
}
