package openstack

import (
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/config"
)

// resourceMap is the mapping of resources we support to their basic
// operations. This makes it easy to implement new resource types.
var resourceMap *resource.Map

func init() {
	resourceMap = &resource.Map{
		Mapping: map[string]resource.Resource{
			"openstack_compute": resource.Resource{
				ConfigValidator: &config.Validator{
					Required: []string{
						"vpc_id",
						"route.*.cidr_block",
					},
					Optional: []string{
						"route.*.gateway_id",
						"route.*.instance_id",
					},
				},
				Create:          resource_openstack_compute_create,
				Destroy:         resource_openstack_compute_destroy,
				Diff:            resource_openstack_compute_diff,
				Update:          resource_openstack_compute_update,
				Refresh:         resource_openstack_compute_refresh,
			},
		},
	}
}
