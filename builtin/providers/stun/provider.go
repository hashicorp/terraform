package stun

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		DataSourcesMap: map[string]*schema.Resource{
			"stun": dataSourceStun(),
		},
		ResourcesMap: map[string]*schema.Resource{
			"stun": schema.DataSourceResourceShim(
				"stun",
				dataSourceStun(),
			),
		},
	}
}
