package main

import (
	"github.com/hashicorp/terraform/builtin/providers/aws"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(new(aws.ResourceProvider))
}
