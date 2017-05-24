package main

import (
	"github.com/r3labs/terraform/builtin/providers/dns"
	"github.com/r3labs/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: dns.Provider,
	})
}
