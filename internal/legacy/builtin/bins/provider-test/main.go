package main

import (
	"github.com/hashicorp/terraform/internal/legacy/builtin/providers/test"
	"github.com/hashicorp/terraform/internal/legacy/terraform"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: func() terraform.ResourceProvider {
			return test.Provider()
		},
	})
}
