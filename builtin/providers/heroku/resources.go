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
			"heroku_app": resource.Resource{
				ConfigValidator: resource_heroku_app_validation(),
				Create:          resource_heroku_app_create,
				Destroy:         resource_heroku_app_destroy,
				Diff:            resource_heroku_app_diff,
				Refresh:         resource_heroku_app_refresh,
				Update:          resource_heroku_app_update,
			},
		},
	}
}
