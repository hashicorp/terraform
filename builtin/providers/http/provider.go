package http

import (
	"github.com/r3labs/terraform/helper/schema"
	"github.com/r3labs/terraform/terraform"
)

func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{},

		DataSourcesMap: map[string]*schema.Resource{
			"http": dataSource(),
		},

		ResourcesMap: map[string]*schema.Resource{},
	}
}
