package rancher

import (
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
				DefaultFunc: schema.EnvDefaultFunc("RANCHER_URL", nil),
				Description: descriptions["api_url"],
			},
			"access_key": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("RANCHER_ACCESS_KEY", ""),
				Description: descriptions["access_key"],
			},
			"secret_key": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("RANCHER_SECRET_KEY", ""),
				Description: descriptions["secret_key"],
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"rancher_environment":         resourceRancherEnvironment(),
			"rancher_registration_token":  resourceRancherRegistrationToken(),
			"rancher_registry":            resourceRancherRegistry(),
			"rancher_registry_credential": resourceRancherRegistryCredential(),
			"rancher_stack":               resourceRancherStack(),
		},

		ConfigureFunc: providerConfigure,
	}
}

var descriptions map[string]string

func init() {
	descriptions = map[string]string{
		"access_key": "API Key used to authenticate with the rancher server",

		"secret_key": "API secret used to authenticate with the rancher server",

		"api_url": "The URL to the rancher API",
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := &Config{
		APIURL:    d.Get("api_url").(string) + "/v1",
		AccessKey: d.Get("access_key").(string),
		SecretKey: d.Get("secret_key").(string),
	}

	_, err := config.GlobalClient()

	return config, err
}
