package akamai

import (
	"github.com/akamai-open/AkamaiOPEN-edgegrid-golang"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"edgerc": &schema.Schema{
				Optional: true,
				Type:     schema.TypeString,
			},
			"fastdns_section": &schema.Schema{
				Optional: true,
				Type:     schema.TypeString,
				Default:  "default",
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"akamai_fastdns_record": resourceFastDnsRecord(),
		},
		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	edgerc := d.Get("edgerc").(string)
	section := d.Get("fastdns_section").(string)

	edge_config, err := edgegrid.Init(edgerc, section)
	if err != nil {
		return nil, err
	}

	fastDnsClient, err := edgegrid.New(nil, edge_config)
	if err != nil {
		return nil, err
	}

	zone := NewZone(fastDnsClient, "")

	config := Config{ClientFastDns: &zone}

	return &config, nil
}
