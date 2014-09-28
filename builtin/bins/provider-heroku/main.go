package main

import (
	"github.com/hashicorp/terraform/builtin/providers/heroku"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: heroku.Provider,
	})
}
