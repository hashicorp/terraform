package opsgenie

import (
	"log"

	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// Provider represents a resource provider in Terraform
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"api_key": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("OPSGENIE_API_KEY", nil),
			},
		},

		DataSourcesMap: map[string]*schema.Resource{},

		ResourcesMap: map[string]*schema.Resource{
			"opsgenie_user": resourceOpsGenieUser(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func (c *Config) validate() error {
	var err *multierror.Error

	if c.ApiKey == "" {
		err = multierror.Append(err, fmt.Errorf("API Key must be configured for the OpsGenie provider"))
	}

	return err.ErrorOrNil()
}

func providerConfigure(data *schema.ResourceData) (interface{}, error) {
	log.Println("[INFO] Initializing OpsGenie client")
	config := Config{
		ApiKey: data.Get("api_key").(string),
	}

	if err := config.validate(); err != nil {
		return nil, err
	}

	return config.Client()
}
