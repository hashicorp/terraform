package google

import (
	"github.com/hashicorp/terraform/helper/schema"
)

// Provider returns a terraform.ResourceProvider.
func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"account_file": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"client_secrets_file": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"google_compute_instance": resourceComputeInstance(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := Config{
		AccountFile:       d.Get("account_file").(string),
		ClientSecretsFile: d.Get("client_secrets_file").(string),
	}

	if err := config.loadAndValidate(); err != nil {
		return nil, err
	}

	return nil, nil
}
