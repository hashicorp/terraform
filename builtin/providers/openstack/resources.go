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

			"openstack_network": resource.Resource{
				ConfigValidator: &config.Validator{
					Required: []string{
						"name",
						"subnet.*.cidr",
						"subnet.*.ip_version",
					},
					Optional: []string{
						"subnet.*.name",
						"subnet.*.enable_dhcp",
					},
				},
				Create:  resource_openstack_network_create,
				Destroy: resource_openstack_network_destroy,
				Diff:    resource_openstack_network_diff,
				Refresh: resource_openstack_network_refresh,
			},

			"openstack_security_group": resource.Resource{
				ConfigValidator: &config.Validator{
					Required: []string{
						"name",
						"rule.*.direction",
						"rule.*.remote_ip_prefix",
					},
					Optional: []string{
						"description",
						"rule.*.port_range_min",
						"rule.*.port_range_max",
						"rule.*.protocol",
					},
				},
				Create:  resource_openstack_security_group_create,
				Destroy: resource_openstack_security_group_destroy,
				Diff:    resource_openstack_security_group_diff,
				Refresh: resource_openstack_security_group_refresh,
			},
		},
	}
}
