package tutum

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// Provider returns a schema.Provider for TutumCloud
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"user": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("TUTUM_USER", nil),
				Description: "The user to authenticate as.",
			},
			"apikey": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("TUTUM_APIKEY", nil),
				Description: "API key used to authenticate the user.",
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"tutum_node_cluster": resourceTutumNodeCluster(),
			"tutum_service":      resourceTutumService(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := Config{
		User:   d.Get("user").(string),
		ApiKey: d.Get("apikey").(string),
	}
	return config, config.Load()
}
