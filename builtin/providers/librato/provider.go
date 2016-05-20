package librato

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"github.com/henrikhodne/go-librato/librato"
)

// Provider returns a schema.Provider for Librato.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"email": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("LIBRATO_EMAIL", nil),
				Description: "The email address for the Librato account.",
			},

			"token": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("LIBRATO_TOKEN", nil),
				Description: "The auth token for the Librato account.",
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"librato_space":       resourceLibratoSpace(),
			"librato_space_chart": resourceLibratoSpaceChart(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	client := librato.NewClient(d.Get("email").(string), d.Get("token").(string))

	return client, nil
}
