package packet

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// Provider returns a schema.Provider for Packet.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"auth_token": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("PACKET_AUTH_TOKEN", nil),
				Description: "The API auth key for API operations.",
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"packet_device":  resourcePacketDevice(),
			"packet_ssh_key": resourcePacketSSHKey(),
			"packet_project": resourcePacketProject(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := Config{
		AuthToken: d.Get("auth_token").(string),
	}

	return config.Client(), nil
}
