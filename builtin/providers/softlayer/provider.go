package softlayer

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"username": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("SOFTLAYER_USERNAME", nil),
				Description: "The user name for SoftLayer API operations.",
			},
			"api_key": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("SOFTLAYER_API_KEY", nil),
				Description: "The API key for SoftLayer API operations.",
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"softlayer_virtual_guest":                           resourceSoftLayerVirtualGuest(),
			"softlayer_ssh_key":                                 resourceSoftLayerSSHKey(),
			"softlayer_dns_domain_record":                       resourceSoftLayerDnsDomainResourceRecord(),
			"softlayer_dns_domain":                              resourceSoftLayerDnsDomain(),
			"softlayer_network_application_delivery_controller": resourceSoftLayerNetworkApplicationDeliveryController(),
			"softlayer_network_loadbalancer_virtualipaddress":   resourceSoftLayerNetworkLoadBalancerVirtualIpAddress(),
			"softlayer_network_loadbalancer_service":            resourceSoftLayerNetworkLoadBalancerService(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := Config{
		Username: d.Get("username").(string),
		ApiKey:   d.Get("api_key").(string),
	}

	return config.Client()
}
