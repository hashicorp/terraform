package openstack

import (
	"github.com/hashicorp/terraform/helper/config"
	"github.com/hashicorp/terraform/helper/resource"
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
						"image_ref",
						"flavor_ref",
					},
					Optional: []string{
						"name",
						"networks.*",
						"security_groups.*",
					},
				},
				Create:  resource_openstack_compute_create,
				Destroy: resource_openstack_compute_destroy,
				Diff:    resource_openstack_compute_diff,
				Update:  resource_openstack_compute_update,
				Refresh: resource_openstack_compute_refresh,
			},
		},
	}
}
