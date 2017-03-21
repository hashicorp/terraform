package spotinst

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"email": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("SPOTINST_EMAIL", ""),
				Description: "Spotinst Email",
			},

			"password": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("SPOTINST_PASSWORD", ""),
				Description: "Spotinst Password",
			},

			"client_id": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("SPOTINST_CLIENT_ID", ""),
				Description: "Spotinst OAuth Client ID",
			},

			"client_secret": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("SPOTINST_CLIENT_SECRET", ""),
				Description: "Spotinst OAuth Client Secret",
			},

			"token": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("SPOTINST_TOKEN", ""),
				Description: "Spotinst Personal API Access Token",
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"spotinst_aws_group":    resourceSpotinstAwsGroup(),
			"spotinst_subscription": resourceSpotinstSubscription(),
			"spotinst_healthcheck":  resourceSpotinstHealthCheck(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := Config{
		Email:        d.Get("email").(string),
		Password:     d.Get("password").(string),
		ClientID:     d.Get("client_id").(string),
		ClientSecret: d.Get("client_secret").(string),
		Token:        d.Get("token").(string),
	}
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return config.Client()
}
