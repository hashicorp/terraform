package rundeck

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"

	"github.com/apparentlymart/go-rundeck-api/rundeck"
)

func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"url": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("RUNDECK_URL", nil),
				Description: "URL of the root of the target Rundeck server.",
			},
			"auth_token": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("RUNDECK_AUTH_TOKEN", nil),
				Description: "Auth token to use with the Rundeck API.",
			},
			"allow_unverified_ssl": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "If set, the Rundeck client will permit unverifiable SSL certificates.",
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"rundeck_project":     resourceRundeckProject(),
			"rundeck_job":         resourceRundeckJob(),
			"rundeck_private_key": resourceRundeckPrivateKey(),
			"rundeck_public_key":  resourceRundeckPublicKey(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := &rundeck.ClientConfig{
		BaseURL:            d.Get("url").(string),
		AuthToken:          d.Get("auth_token").(string),
		AllowUnverifiedSSL: d.Get("allow_unverified_ssl").(bool),
	}

	return rundeck.NewClient(config)
}
