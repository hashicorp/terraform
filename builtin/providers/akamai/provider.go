package akamai

import (
	"github.com/akamai/AkamaiOPEN-edgegrid-golang/configdns-v1"
	"github.com/akamai/AkamaiOPEN-edgegrid-golang/edgegrid"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"github.com/akamai/AkamaiOPEN-edgegrid-golang/papi-v1"
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
	err := getConfigDNSV1Service(d)
	if err != nil {
		return nil, err
	}

	err = getPAPIV1Service(d)
	if err != nil {
		return nil, err
	}

	return &Config{}, nil
}

func getConfigDNSV1Service(d *schema.ResourceData) error {
	edgerc := d.Get("edgerc").(string)
	section := d.Get("fastdns_section").(string)

	fastDnsConfig, err := edgegrid.InitEdgeRc(edgerc, section)
	if err != nil {
		return err
	}

	dns.Init(fastDnsConfig)

	return nil
}

func getPAPIV1Service(d *schema.ResourceData) error {
	edgerc := d.Get("edgerc").(string)
	section := d.Get("papi_section").(string)

	papiConfig, err := edgegrid.InitEdgeRc(edgerc, section)
	if err != nil {
		return err
	}

	papi.Init(papiConfig)

	return nil
}
