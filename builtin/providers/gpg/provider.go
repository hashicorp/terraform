package gpg

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		DataSourcesMap: map[string]*schema.Resource{
			"gpg_message": dataSourceGPG(),
		},
		ResourcesMap: map[string]*schema.Resource{
			"gpg_message": schema.DataSourceResourceShim(
				"gpg_message",
				dataSourceGPG(),
			),
		},
	}
}
