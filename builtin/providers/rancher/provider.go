package rancher

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"server_url": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("RANCHER_URL", nil),
				Description: "Rancher server URL",
			},
			"access_key": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("RANCHER_ACCESS_KEY", nil),
				Description: "Rancher API access key",
			},
			"secret_key": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("RANCHER_SECRET_KEY", nil),
				Description: "Rancher API secret key",
			},
		},

		resourcesmap: map[string]*schema.Resource{
			"rancher_environment": resourceRancherEnvironment(),
		},

		configurefunc: providerconfigure,
	}
}

func providerConfigure(data *schema.ResourceData) (interface{}, error) {
	config := Config{
		ServerUrl: data.Get("server_url").(string),
		AccessKey: data.Get("access_key").(string),
		SecretKey: data.Get("secret_key").(string),
	}

	client, err := config.Client()
	if err != nil {
		return nil, fmt.Errorf("Error initializing Postgresql client: %s", err)
	}

	return client, nil
}
