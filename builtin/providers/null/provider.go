package null

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
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
