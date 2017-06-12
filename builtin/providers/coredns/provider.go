package coredns

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"etcd_endpoints": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("COREDNS_ETCD_ENDPOINTS", nil),
				Description: "CoreDNS etcd endpoints.",
			},
			"zones": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("COREDNS_ZONES", nil),
				Description: "CoreDNS managed zones.",
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"coredns_record": resourceCorednsRecord(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := Config{
		EtcdEndpoints: d.Get("etcd_endpoints").(string),
		Zones:         d.Get("zones").(string),
	}
	return config.newDNSOp()
}
