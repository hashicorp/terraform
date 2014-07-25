package cloudflare

import (
	"github.com/hashicorp/terraform/helper/resource"
)

// resourceMap is the mapping of resources we support to their basic
// operations. This makes it easy to implement new resource types.
var resourceMap *resource.Map

func init() {
	resourceMap = &resource.Map{
		Mapping: map[string]resource.Resource{
			"cloudflare_record": resource.Resource{
				ConfigValidator: resource_cloudflare_record_validation(),
				Create:          resource_cloudflare_record_create,
				Destroy:         resource_cloudflare_record_destroy,
				Diff:            resource_cloudflare_record_diff,
				Update:          resource_cloudflare_record_update,
				Refresh:         resource_cloudflare_record_refresh,
			},
		},
	}
}
