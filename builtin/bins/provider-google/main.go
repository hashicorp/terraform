package main

import (
	"github.com/hashicorp/terraform/builtin/providers/google"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: google.Provider,
	})
}
