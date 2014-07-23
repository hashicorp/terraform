package main

import (
	"github.com/hashicorp/terraform/builtin/providers/dnsimple"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(new(dnsimple.ResourceProvider))
}
