package main

import (
	"github.com/hashicorp/terraform/builtin/providers/mailgun"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: mailgun.Provider,
	})
}
