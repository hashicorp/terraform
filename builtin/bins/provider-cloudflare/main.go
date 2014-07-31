package main

import (
	"github.com/hashicorp/terraform/builtin/providers/cloudflare"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(new(cloudflare.ResourceProvider))
}
