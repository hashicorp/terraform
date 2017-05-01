package heroku

import (
	"fmt"
	"log"
	"strings"

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
			"heroku_addon":             resourceHerokuAddon(),
			"heroku_app":               resourceHerokuApp(),
			"heroku_app_feature":       resourceHerokuAppFeature(),
			"heroku_cert":              resourceHerokuCert(),
			"heroku_domain":            resourceHerokuDomain(),
			"heroku_drain":             resourceHerokuDrain(),
			"heroku_pipeline":          resourceHerokuPipeline(),
			"heroku_pipeline_coupling": resourceHerokuPipelineCoupling(),
			"heroku_space":             resourceHerokuSpace(),
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

func buildCompositeID(a, b string) string {
	return fmt.Sprintf("%s:%s", a, b)
}

func parseCompositeID(id string) (string, string) {
	parts := strings.SplitN(id, ":", 2)
	return parts[0], parts[1]
}
