package aws

import (
	"github.com/hashicorp/terraform/helper/resource"
)

// resourceMap is the mapping of resources we support to their basic
// operations. This makes it easy to implement new resource types.
var resourceMap *resource.Map

func init() {
	resourceMap = &resource.Map{
		Mapping: map[string]resource.Resource{
			"aws_elb": resource.Resource{
				Create:  resource_aws_elb_create,
				Update:  resource_aws_elb_update,
				Destroy: resource_aws_elb_destroy,
				Diff:    resource_aws_elb_diff,
				Refresh: resource_aws_elb_refresh,
			},

			"aws_instance": resource.Resource{
				Create:  resource_aws_instance_create,
				Destroy: resource_aws_instance_destroy,
				Diff:    resource_aws_instance_diff,
				Refresh: resource_aws_instance_refresh,
			},
		},
	}
}
