package heroku

import (
	"log"
	"os"

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
				DefaultFunc: envDefaultFunc("HEROKU_EMAIL"),
			},

			"api_key": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: envDefaultFunc("HEROKU_API_KEY"),
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"heroku_app":    resourceHerokuApp(),
			"heroku_addon":  resourceHerokuAddon(),
			"heroku_domain": resourceHerokuDomain(),
			"heroku_drain":  resourceHerokuDrain(),
			"heroku_cert":  resourceHerokuCert(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func envDefaultFunc(k string) schema.SchemaDefaultFunc {
	return func() (interface{}, error) {
		if v := os.Getenv(k); v != "" {
			return v, nil
		}

		return nil, nil
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
