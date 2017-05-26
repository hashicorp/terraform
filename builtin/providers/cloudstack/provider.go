package cloudstack

import (
	"errors"

	"github.com/go-ini/ini"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"api_url": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				DefaultFunc:   schema.EnvDefaultFunc("CLOUDSTACK_API_URL", nil),
				ConflictsWith: []string{"config", "profile"},
			},

			"api_key": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				DefaultFunc:   schema.EnvDefaultFunc("CLOUDSTACK_API_KEY", nil),
				ConflictsWith: []string{"config", "profile"},
			},

			"secret_key": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				DefaultFunc:   schema.EnvDefaultFunc("CLOUDSTACK_SECRET_KEY", nil),
				ConflictsWith: []string{"config", "profile"},
			},

			"config": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"api_url", "api_key", "secret_key"},
			},

			"profile": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"api_url", "api_key", "secret_key"},
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
			"cloudstack_affinity_group":       resourceCloudStackAffinityGroup(),
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
			"cloudstack_private_gateway":      resourceCloudStackPrivateGateway(),
			"cloudstack_secondary_ipaddress":  resourceCloudStackSecondaryIPAddress(),
			"cloudstack_security_group":       resourceCloudStackSecurityGroup(),
			"cloudstack_security_group_rule":  resourceCloudStackSecurityGroupRule(),
			"cloudstack_ssh_keypair":          resourceCloudStackSSHKeyPair(),
			"cloudstack_static_nat":           resourceCloudStackStaticNAT(),
			"cloudstack_static_route":         resourceCloudStackStaticRoute(),
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
	apiURL, apiURLOK := d.GetOk("api_url")
	apiKey, apiKeyOK := d.GetOk("api_key")
	secretKey, secretKeyOK := d.GetOk("secret_key")
	config, configOK := d.GetOk("config")
	profile, profileOK := d.GetOk("profile")

	switch {
	case apiURLOK, apiKeyOK, secretKeyOK:
		if !(apiURLOK && apiKeyOK && secretKeyOK) {
			return nil, errors.New("'api_url', 'api_key' and 'secret_key' should all have values")
		}
	case configOK, profileOK:
		if !(configOK && profileOK) {
			return nil, errors.New("'config' and 'profile' should both have a value")
		}
	default:
		return nil, errors.New(
			"either 'api_url', 'api_key' and 'secret_key' or 'config' and 'profile' should have values")
	}

	if configOK && profileOK {
		cfg, err := ini.Load(config.(string))
		if err != nil {
			return nil, err
		}

		section, err := cfg.GetSection(profile.(string))
		if err != nil {
			return nil, err
		}

		apiURL = section.Key("url").String()
		apiKey = section.Key("apikey").String()
		secretKey = section.Key("secretkey").String()
	}

	cfg := Config{
		APIURL:      apiURL.(string),
		APIKey:      apiKey.(string),
		SecretKey:   secretKey.(string),
		HTTPGETOnly: d.Get("http_get_only").(bool),
		Timeout:     int64(d.Get("timeout").(int)),
	}

	return cfg.NewClient()
}
