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
			"digitalocean_droplet": resource.Resource{
				ConfigValidator: resource_digitalocean_droplet_validation(),
				Create:          resource_digitalocean_droplet_create,
				Destroy:         resource_digitalocean_droplet_destroy,
				Diff:            resource_digitalocean_droplet_diff,
				Refresh:         resource_digitalocean_droplet_refresh,
				Update:          resource_digitalocean_droplet_update,
			},
		},
	}
}
