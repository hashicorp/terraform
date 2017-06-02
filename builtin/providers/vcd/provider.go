package vcd

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"user": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("VCD_USER", nil),
				Description: "The user name for vcd API operations.",
			},

			"password": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("VCD_PASSWORD", nil),
				Description: "The user password for vcd API operations.",
			},

			"org": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("VCD_ORG", nil),
				Description: "The vcd org for API operations",
			},

			"url": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("VCD_URL", nil),
				Description: "The vcd url for vcd API operations.",
			},

			"vdc": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("VCD_VDC", ""),
				Description: "The name of the VDC to run operations on",
			},

			"maxRetryTimeout": &schema.Schema{
				Type:        schema.TypeInt,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("VCD_MAX_RETRY_TIMEOUT", 60),
				Description: "Max num seconds to wait for successful response when operating on resources within vCloud (defaults to 60)",
			},

			"allow_unverified_ssl": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("VCD_ALLOW_UNVERIFIED_SSL", false),
				Description: "If set, VCDClient will permit unverifiable SSL certificates.",
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"vcd_network":         resourceVcdNetwork(),
			"vcd_vapp":            resourceVcdVApp(),
			"vcd_firewall_rules":  resourceVcdFirewallRules(),
			"vcd_dnat":            resourceVcdDNAT(),
			"vcd_snat":            resourceVcdSNAT(),
			"vcd_edgegateway_vpn": resourceVcdEdgeGatewayVpn(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := Config{
		User:            d.Get("user").(string),
		Password:        d.Get("password").(string),
		Org:             d.Get("org").(string),
		Href:            d.Get("url").(string),
		VDC:             d.Get("vdc").(string),
		MaxRetryTimeout: d.Get("maxRetryTimeout").(int),
		InsecureFlag:    d.Get("allow_unverified_ssl").(bool),
	}

	return config.Client()
}
