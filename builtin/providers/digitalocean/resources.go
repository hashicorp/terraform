package digitalocean

import (
	"github.com/hashicorp/terraform/helper/resource"
)

// resourceMap is the mapping of resources we support to their basic
// operations. This makes it easy to implement new resource types.
var resourceMap *resource.Map

func init() {
	resourceMap = &resource.Map{
		Mapping: map[string]resource.Resource{
			"digitalocean_domain": resource.Resource{
				ConfigValidator: resource_digitalocean_domain_validation(),
				Create:          resource_digitalocean_domain_create,
				Destroy:         resource_digitalocean_domain_destroy,
				Diff:            resource_digitalocean_domain_diff,
				Refresh:         resource_digitalocean_domain_refresh,
			},

			"digitalocean_droplet": resource.Resource{
				ConfigValidator: resource_digitalocean_droplet_validation(),
				Create:          resource_digitalocean_droplet_create,
				Destroy:         resource_digitalocean_droplet_destroy,
				Diff:            resource_digitalocean_droplet_diff,
				Refresh:         resource_digitalocean_droplet_refresh,
				Update:          resource_digitalocean_droplet_update,
			},

			"digitalocean_record": resource.Resource{
				ConfigValidator: resource_digitalocean_record_validation(),
				Create:          resource_digitalocean_record_create,
				Destroy:         resource_digitalocean_record_destroy,
				Update:          resource_digitalocean_record_update,
				Diff:            resource_digitalocean_record_diff,
				Refresh:         resource_digitalocean_record_refresh,
			},
		},
	}
}
