package cloudstack

import (
	"os"

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
				DefaultFunc: envDefaultFunc("CLOUDSTACK_API_URL"),
			},

			"api_key": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: envDefaultFunc("CLOUDSTACK_API_KEY"),
			},

			"secret_key": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: envDefaultFunc("CLOUDSTACK_SECRET_KEY"),
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"cloudstack_disk":             resourceCloudStackDisk(),
			"cloudstack_firewall":         resourceCloudStackFirewall(),
			"cloudstack_instance":         resourceCloudStackInstance(),
			"cloudstack_ipaddress":        resourceCloudStackIPAddress(),
			"cloudstack_network":          resourceCloudStackNetwork(),
			"cloudstack_network_acl":      resourceCloudStackNetworkACL(),
			"cloudstack_network_acl_rule": resourceCloudStackNetworkACLRule(),
			"cloudstack_nic":              resourceCloudStackNIC(),
			"cloudstack_port_forward":     resourceCloudStackPortForward(),
			"cloudstack_vpc":              resourceCloudStackVPC(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := Config{
		ApiURL:    d.Get("api_url").(string),
		ApiKey:    d.Get("api_key").(string),
		SecretKey: d.Get("secret_key").(string),
	}

	return config.NewClient()
}

func envDefaultFunc(k string) schema.SchemaDefaultFunc {
	return func() (interface{}, error) {
		if v := os.Getenv(k); v != "" {
			return v, nil
		}

		return nil, nil
	}
}
