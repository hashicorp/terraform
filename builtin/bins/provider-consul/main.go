package main

import (
	"github.com/hashicorp/terraform/builtin/providers/consul"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(new(consul.ResourceProvider))
}
