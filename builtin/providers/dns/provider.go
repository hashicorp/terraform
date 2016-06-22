package dns

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{},

		ResourcesMap: map[string]*schema.Resource{
			"dns_a_record": schema.DataSourceResourceShim(
				"dns_a_record",
				dataSourceDnsARecord(),
			),
		},
		DataSourcesMap: map[string]*schema.Resource{
			"dns_a_record":     dataSourceDnsARecord(),
			"dns_cname_record": dataSourceDnsCnameRecord(),
			"dns_text_record":  dataSourceDnsTxtRecord(),
		},
	}
}
