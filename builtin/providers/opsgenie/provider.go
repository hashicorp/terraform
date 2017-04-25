package opsgenie

import (
	"log"

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

		DataSourcesMap: map[string]*schema.Resource{
			"opsgenie_user": dataSourceOpsGenieUser(),
		},

		ResourcesMap: map[string]*schema.Resource{
			"opsgenie_team": resourceOpsGenieTeam(),
			"opsgenie_user": resourceOpsGenieUser(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(data *schema.ResourceData) (interface{}, error) {
	log.Println("[INFO] Initializing OpsGenie client")

	config := Config{
		ApiKey: data.Get("api_key").(string),
	}

	return config.Client()
}
