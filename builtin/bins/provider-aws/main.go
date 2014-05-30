package main

import (
	"github.com/hashicorp/terraform/plugin"
	"github.com/hashicorp/terraform/builtin/providers/aws"
)

func main() {
	plugin.Serve(new(aws.ResourceProvider))
}
