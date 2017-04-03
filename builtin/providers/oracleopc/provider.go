package opc

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
				DefaultFunc: schema.EnvDefaultFunc("OPC_USERNAME", nil),
				Description: "The user name for OPC API operations.",
			},

			"password": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("OPC_PASSWORD", nil),
				Description: "The user password for OPC API operations.",
			},

			"identityDomain": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("OPC_IDENTITY_DOMAIN", nil),
				Description: "The OPC identity domain for API operations",
			},

			"endpoint": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("OPC_ENDPOINT", nil),
				Description: "The HTTP endpoint for OPC API operations.",
			},

			"maxRetryTimeout": &schema.Schema{
				Type:        schema.TypeInt,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("OPC_MAX_RETRY_TIMEOUT", 3000),
				Description: "Max num seconds to wait for successful response when operating on resources within OPC (defaults to 3000)",
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"opc_compute_storage_volume":       resourceStorageVolume(),
			"opc_compute_instance":             resourceInstance(),
			"opc_compute_ssh_key":              resourceSSHKey(),
			"opc_compute_security_application": resourceSecurityApplication(),
			"opc_compute_security_list":        resourceSecurityList(),
			"opc_compute_security_ip_list":     resourceSecurityIPList(),
			"opc_compute_ip_reservation":       resourceIPReservation(),
			"opc_compute_ip_association":       resourceIPAssociation(),
			"opc_compute_security_rule":        resourceSecurityRule(),
			"opc_compute_security_association": resourceSecurityAssociation(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := Config{
		User:            d.Get("user").(string),
		Password:        d.Get("password").(string),
		IdentityDomain:  d.Get("identityDomain").(string),
		Endpoint:        d.Get("endpoint").(string),
		MaxRetryTimeout: d.Get("maxRetryTimeout").(int),
	}

	return config.Client()
}
