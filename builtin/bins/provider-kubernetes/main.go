package main

import (
	"github.com/hashicorp/terraform/builtin/providers/kubernetes"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: kubernetes.Provider,
	})
}
