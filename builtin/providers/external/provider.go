package external

import (
	"github.com/r3labs/terraform/helper/schema"
	"github.com/r3labs/terraform/terraform"
)

func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		DataSourcesMap: map[string]*schema.Resource{
			"external": dataSource(),
		},
		ResourcesMap: map[string]*schema.Resource{},
	}
}
