package heroku

import (
	"github.com/hashicorp/terraform/helper/resource"
)

// resourceMap is the mapping of resources we support to their basic
// operations. This makes it easy to implement new resource types.
var resourceMap *resource.Map

func init() {
	resourceMap = &resource.Map{
		Mapping: map[string]resource.Resource{
			"heroku_addon": resource.Resource{
				ConfigValidator: resource_heroku_addon_validation(),
				Create:          resource_heroku_addon_create,
				Destroy:         resource_heroku_addon_destroy,
				Diff:            resource_heroku_addon_diff,
				Refresh:         resource_heroku_addon_refresh,
				Update:          resource_heroku_addon_update,
			},

			"heroku_app": resource.Resource{
				ConfigValidator: resource_heroku_app_validation(),
				Create:          resource_heroku_app_create,
				Destroy:         resource_heroku_app_destroy,
				Diff:            resource_heroku_app_diff,
				Refresh:         resource_heroku_app_refresh,
				Update:          resource_heroku_app_update,
			},

			"heroku_domain": resource.Resource{
				ConfigValidator: resource_heroku_domain_validation(),
				Create:          resource_heroku_domain_create,
				Destroy:         resource_heroku_domain_destroy,
				Diff:            resource_heroku_domain_diff,
				Refresh:         resource_heroku_domain_refresh,
			},

			"heroku_drain": resource.Resource{
				ConfigValidator: resource_heroku_drain_validation(),
				Create:          resource_heroku_drain_create,
				Destroy:         resource_heroku_drain_destroy,
				Diff:            resource_heroku_drain_diff,
				Refresh:         resource_heroku_drain_refresh,
			},
		},
	}
}
