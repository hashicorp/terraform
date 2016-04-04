package cloudstack

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"api_url": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("CLOUDSTACK_API_URL", nil),
			},

			"api_key": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("CLOUDSTACK_API_KEY", nil),
			},

			"secret_key": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("CLOUDSTACK_SECRET_KEY", nil),
			},

			"http_get_only": &schema.Schema{
				Type:        schema.TypeBool,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("CLOUDSTACK_HTTP_GET_ONLY", false),
			},

			"timeout": &schema.Schema{
				Type:        schema.TypeInt,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("CLOUDSTACK_TIMEOUT", 900),
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"cloudstack_disk":                 resourceCloudStackDisk(),
			"cloudstack_egress_firewall":      resourceCloudStackEgressFirewall(),
			"cloudstack_firewall":             resourceCloudStackFirewall(),
			"cloudstack_instance":             resourceCloudStackInstance(),
			"cloudstack_ipaddress":            resourceCloudStackIPAddress(),
			"cloudstack_loadbalancer_rule":    resourceCloudStackLoadBalancerRule(),
			"cloudstack_network":              resourceCloudStackNetwork(),
			"cloudstack_network_acl":          resourceCloudStackNetworkACL(),
			"cloudstack_network_acl_rule":     resourceCloudStackNetworkACLRule(),
			"cloudstack_nic":                  resourceCloudStackNIC(),
			"cloudstack_port_forward":         resourceCloudStackPortForward(),
			"cloudstack_secondary_ipaddress":  resourceCloudStackSecondaryIPAddress(),
			"cloudstack_ssh_keypair":          resourceCloudStackSSHKeyPair(),
			"cloudstack_static_nat":           resourceCloudStackStaticNAT(),
			"cloudstack_template":             resourceCloudStackTemplate(),
			"cloudstack_vpc":                  resourceCloudStackVPC(),
			"cloudstack_vpn_connection":       resourceCloudStackVPNConnection(),
			"cloudstack_vpn_customer_gateway": resourceCloudStackVPNCustomerGateway(),
			"cloudstack_vpn_gateway":          resourceCloudStackVPNGateway(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := Config{
		APIURL:      d.Get("api_url").(string),
		APIKey:      d.Get("api_key").(string),
		SecretKey:   d.Get("secret_key").(string),
		HTTPGETOnly: d.Get("http_get_only").(bool),
		Timeout:     int64(d.Get("timeout").(int)),
	}

	return config.NewClient()
}
