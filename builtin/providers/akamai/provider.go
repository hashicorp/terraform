package akamai

import (
	"github.com/akamai-open/AkamaiOPEN-edgegrid-golang/edgegrid"
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
			"akamai_fastdns_record": resourceFastDNSRecord(),
		},
		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	configDnsV1Service, err := getConfigDnsV1Service(d)
	if err != nil {
		return nil, err
	}

	config := &Config{}
	config.ConfigDNSV1Service = configDnsV1Service

	return config, nil
}

func getConfigDNSV1Service(d *schema.ResourceData) (*edgegrid.ConfigDNSV1Service, error) {
	edgerc := d.Get("edgerc").(string)
	section := d.Get("fastdns_section").(string)

	fastDnsConfig, err := edgegrid.Init(edgerc, section)
	if err != nil {
		return nil, err
	}

	fastDnsClient, err := edgegrid.NewClient(nil, &fastDnsConfig)
	if err != nil {
		return nil, err
	}

	client := edgegrid.NewConfigDNSV1Service(fastDnsClient, &fastDnsConfig)

	return client, nil
}
