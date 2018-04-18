package cidr

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		DataSourcesMap: map[string]*schema.Resource{
			"cidr_network": dataSourceNetwork(),
			"cidr_subnet":  dataSourceSubnet(),
		},
	}
}
