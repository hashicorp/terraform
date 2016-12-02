package nomad

import (
	"fmt"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"address": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("NOMAD_ADDR", nil),
				Description: "URL of the root of the target Nomad agent.",
			},

			"region": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("NOMAD_REGION", ""),
				Description: "Region of the target Nomad agent.",
			},
		},

		ConfigureFunc: providerConfigure,

		ResourcesMap: map[string]*schema.Resource{
			"nomad_job": resourceJob(),
		},
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := api.DefaultConfig()
	config.Address = d.Get("address").(string)
	config.Region = d.Get("region").(string)

	client, err := api.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to configure Nomad API: %s", err)
	}

	return client, nil
}
