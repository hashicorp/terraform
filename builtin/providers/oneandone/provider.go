package oneandone

import (
	"github.com/1and1/oneandone-cloudserver-sdk-go"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"token": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("ONEANDONE_TOKEN", nil),
				Description: "1&1 token for API operations.",
			},
			"retries": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     50,
				DefaultFunc: schema.EnvDefaultFunc("ONEANDONE_RETRIES", nil),
			},
			"endpoint": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     oneandone.BaseUrl,
				DefaultFunc: schema.EnvDefaultFunc("ONEANDONE_ENDPOINT", nil),
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"oneandone_server":            resourceOneandOneServer(),
			"oneandone_firewall_policy":   resourceOneandOneFirewallPolicy(),
			"oneandone_private_network":   resourceOneandOnePrivateNetwork(),
			"oneandone_public_ip":         resourceOneandOnePublicIp(),
			"oneandone_shared_storage":    resourceOneandOneSharedStorage(),
			"oneandone_monitoring_policy": resourceOneandOneMonitoringPolicy(),
			"oneandone_loadbalancer":      resourceOneandOneLoadbalancer(),
			"oneandone_vpn":               resourceOneandOneVPN(),
		},
		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	var endpoint string
	if d.Get("endpoint").(string) != oneandone.BaseUrl {
		endpoint = d.Get("endpoint").(string)
	}
	config := Config{
		Token:    d.Get("token").(string),
		Retries:  d.Get("retries").(int),
		Endpoint: endpoint,
	}
	return config.Client()
}
