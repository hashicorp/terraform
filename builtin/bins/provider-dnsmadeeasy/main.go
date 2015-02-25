package main

import (
	"github.com/hashicorp/terraform/builtin/providers/dnsmadeeasy"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: dnsmadeeasy.Provider,
	})
}
