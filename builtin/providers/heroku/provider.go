package heroku

import (
	"log"

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
				DefaultFunc: schema.EnvDefaultFunc("HEROKU_EMAIL", nil),
			},

			"api_key": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("HEROKU_API_KEY", nil),
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"heroku_app":    resourceHerokuApp(),
			"heroku_addon":  resourceHerokuAddon(),
			"heroku_domain": resourceHerokuDomain(),
			"heroku_drain":  resourceHerokuDrain(),
			"heroku_cert":   resourceHerokuCert(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := Config{
		Email:  d.Get("email").(string),
		APIKey: d.Get("api_key").(string),
	}

	log.Println("[INFO] Initializing Heroku client")
	return config.Client()
}
