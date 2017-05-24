package main

import (
	"github.com/r3labs/terraform/builtin/providers/vault"
	"github.com/r3labs/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: vault.Provider,
	})
}
