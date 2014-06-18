package aws

import (
	"github.com/hashicorp/terraform/diff"
)

var diffBuilder *diff.LazyResourceMap

func init() {
	diffBuilder = &diff.LazyResourceMap{
		Resources: map[string]diff.ResourceBuilderFactory{
			"aws_instance": diffBuilder_aws_instance,
		},
	}
}

func diffBuilder_aws_instance() *diff.ResourceBuilder {
	return &diff.ResourceBuilder{
		CreateComputedAttrs: []string{
			"public_dns",
			"public_ip",
			"private_dns",
			"private_ip",
		},

		RequiresNewAttrs: []string{
			"ami",
			"availability_zone",
			"instance_type",
			"region",
		},
	}
}
