package runscope

import (
	"os"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"access_token": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: envDefaultFunc("RUNSCOPE_ACCESS_TOKEN"),
				Description: "A runscope access token.",
			},
			"api_url": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: envDefaultFunc("RUNSCOPE_API_URL"),
				Description: "A runscope api url i.e. https://api.runscope.com.",
				Default:     "https://api.runscope.com",
			},
		},

		DataSourcesMap: map[string]*schema.Resource{
			"runscope_integration": dataSourceRunscopeIntegration(),
		},

		ResourcesMap: map[string]*schema.Resource{
			"runscope_bucket":      resourceRunscopeBucket(),
			"runscope_test":        resourceRunscopeTest(),
			"runscope_environment": resourceRunscopeEnvironment(),
			"runscope_schedule":    resourceRunscopeSchedule(),
			"runscope_step":        resourceRunscopeStep(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func envDefaultFunc(k string) schema.SchemaDefaultFunc {
	return func() (interface{}, error) {
		if v := os.Getenv(k); v != "" {
			if v == "true" {
				return true, nil
			} else if v == "false" {
				return false, nil
			}
			return v, nil
		}
		return nil, nil
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := Config{
		AccessToken: d.Get("access_token").(string),
		ApiUrl:      d.Get("api_url").(string),
	}
	return config.Client()
}
