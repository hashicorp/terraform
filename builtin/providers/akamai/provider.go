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
			"papi_section": &schema.Schema{
				Optional: true,
				Type:     schema.TypeString,
				Default:  "default",
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"akamai_fastdns_record": resourceFastDNSRecord(),
			"akamai_property":       resourceProperty(),
		},
		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	configDnsV1Service, err := getConfigDNSV1Service(d)
	papiV0Service, err := getPAPIV0Service(d)
	if err != nil {
		return nil, err
	}

	config := &Config{}
	config.ConfigDNSV1Service = configDnsV1Service
	config.PapiV0Service = papiV0Service

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

func getPAPIV0Service(d *schema.ResourceData) (*edgegrid.PapiV0Service, error) {
	edgerc := d.Get("edgerc").(string)
	section := d.Get("papi_section").(string)

	papiConfig, err := edgegrid.Init(edgerc, section)
	if err != nil {
		return nil, err
	}

	papiClient, err := edgegrid.NewClient(nil, &papiConfig)
	if err != nil {
		return nil, err
	}

	client := edgegrid.NewPapiV0Service(papiClient, &papiConfig)

	return client, nil
}
