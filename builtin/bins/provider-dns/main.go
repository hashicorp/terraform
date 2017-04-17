package main

import (
	"github.com/hashicorp/terraform/builtin/providers/dns"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: dns.Provider,
	})
}
