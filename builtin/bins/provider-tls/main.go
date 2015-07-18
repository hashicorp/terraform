package main

import (
	"github.com/hashicorp/terraform/builtin/providers/tls"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: tls.Provider,
	})
}
