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

			"aws_eip": resource.Resource{
				Create:  resource_aws_eip_create,
				Destroy: resource_aws_eip_destroy,
				Diff:    resource_aws_eip_diff,
				Refresh: resource_aws_eip_refresh,
			},

			"aws_instance": resource.Resource{
				Create:  resource_aws_instance_create,
				Destroy: resource_aws_instance_destroy,
				Diff:    resource_aws_instance_diff,
				Refresh: resource_aws_instance_refresh,
			},

			"aws_internet_gateway": resource.Resource{
				Create:  resource_aws_internet_gateway_create,
				Destroy: resource_aws_internet_gateway_destroy,
				Diff:    resource_aws_internet_gateway_diff,
				Refresh: resource_aws_internet_gateway_refresh,
			},

			"aws_security_group": resource.Resource{
				Create:  resource_aws_security_group_create,
				Destroy: resource_aws_security_group_destroy,
				Diff:    resource_aws_security_group_diff,
				Refresh: resource_aws_security_group_refresh,
			},

			"aws_subnet": resource.Resource{
				Create:  resource_aws_subnet_create,
				Destroy: resource_aws_subnet_destroy,
				Diff:    resource_aws_subnet_diff,
				Refresh: resource_aws_subnet_refresh,
			},

			"aws_vpc": resource.Resource{
				Create:  resource_aws_vpc_create,
				Destroy: resource_aws_vpc_destroy,
				Diff:    resource_aws_vpc_diff,
				Refresh: resource_aws_vpc_refresh,
			},
		},
	}
}
