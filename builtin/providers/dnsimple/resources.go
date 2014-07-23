package dnsimple

import (
	"github.com/hashicorp/terraform/helper/resource"
)

// resourceMap is the mapping of resources we support to their basic
// operations. This makes it easy to implement new resource types.
var resourceMap *resource.Map

func init() {
	resourceMap = &resource.Map{
		Mapping: map[string]resource.Resource{
			"dnsimple_record": resource.Resource{
				ConfigValidator: resource_dnsimple_record_validation(),
				Create:          resource_dnsimple_record_create,
				Destroy:         resource_dnsimple_record_destroy,
				Diff:            resource_dnsimple_record_diff,
				Update:          resource_dnsimple_record_update,
				Refresh:         resource_dnsimple_record_refresh,
			},
		},
	}
}
