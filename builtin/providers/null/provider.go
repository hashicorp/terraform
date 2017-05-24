package null

import (
	"github.com/r3labs/terraform/helper/schema"
	"github.com/r3labs/terraform/terraform"
)

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{},

		ResourcesMap: map[string]*schema.Resource{
			"null_resource": resource(),
		},

		DataSourcesMap: map[string]*schema.Resource{
			"null_data_source": dataSource(),
		},
	}
}
