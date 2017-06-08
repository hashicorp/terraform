package opc

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"user": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("OPC_USERNAME", nil),
				Description: "The user name for OPC API operations.",
			},

			"password": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("OPC_PASSWORD", nil),
				Description: "The user password for OPC API operations.",
			},

			"identity_domain": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("OPC_IDENTITY_DOMAIN", nil),
				Description: "The OPC identity domain for API operations",
			},

			"endpoint": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("OPC_ENDPOINT", nil),
				Description: "The HTTP endpoint for OPC API operations.",
			},

			"max_retries": {
				Type:        schema.TypeInt,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("OPC_MAX_RETRIES", 1),
				Description: "Maximum number retries to wait for a successful response when operating on resources within OPC (defaults to 1)",
			},

			"insecure": {
				Type:        schema.TypeBool,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("OPC_INSECURE", false),
				Description: "Skip TLS Verification for self-signed certificates. Should only be used if absolutely required.",
			},
		},

		DataSourcesMap: map[string]*schema.Resource{
			"opc_compute_network_interface": dataSourceNetworkInterface(),
			"opc_compute_vnic":              dataSourceVNIC(),
		},

		ResourcesMap: map[string]*schema.Resource{
			"opc_compute_ip_network":              resourceOPCIPNetwork(),
			"opc_compute_acl":                     resourceOPCACL(),
			"opc_compute_image_list":              resourceOPCImageList(),
			"opc_compute_image_list_entry":        resourceOPCImageListEntry(),
			"opc_compute_instance":                resourceInstance(),
			"opc_compute_ip_address_reservation":  resourceOPCIPAddressReservation(),
			"opc_compute_ip_association":          resourceOPCIPAssociation(),
			"opc_compute_ip_network_exchange":     resourceOPCIPNetworkExchange(),
			"opc_compute_ip_reservation":          resourceOPCIPReservation(),
			"opc_compute_route":                   resourceOPCRoute(),
			"opc_compute_security_application":    resourceOPCSecurityApplication(),
			"opc_compute_security_association":    resourceOPCSecurityAssociation(),
			"opc_compute_security_ip_list":        resourceOPCSecurityIPList(),
			"opc_compute_security_list":           resourceOPCSecurityList(),
			"opc_compute_security_rule":           resourceOPCSecurityRule(),
			"opc_compute_sec_rule":                resourceOPCSecRule(),
			"opc_compute_ssh_key":                 resourceOPCSSHKey(),
			"opc_compute_storage_volume":          resourceOPCStorageVolume(),
			"opc_compute_storage_volume_snapshot": resourceOPCStorageVolumeSnapshot(),
			"opc_compute_vnic_set":                resourceOPCVNICSet(),
			"opc_compute_security_protocol":       resourceOPCSecurityProtocol(),
			"opc_compute_ip_address_prefix_set":   resourceOPCIPAddressPrefixSet(),
			"opc_compute_ip_address_association":  resourceOPCIPAddressAssociation(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := Config{
		User:           d.Get("user").(string),
		Password:       d.Get("password").(string),
		IdentityDomain: d.Get("identity_domain").(string),
		Endpoint:       d.Get("endpoint").(string),
		MaxRetries:     d.Get("max_retries").(int),
		Insecure:       d.Get("insecure").(bool),
	}

	return config.Client()
}
