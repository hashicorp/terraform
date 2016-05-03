package dockercloud

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// Provider returns a schema.Provider for DockerCloud
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"user": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("DOCKERCLOUD_USER", nil),
				Description: "The user to authenticate as.",
			},
			"apikey": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("DOCKERCLOUD_APIKEY", nil),
				Description: "API key used to authenticate the user.",
			},
			"baseurl": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("DOCKERCLOUD_REST_HOST", "https://cloud.docker.com"),
				Description: "API key used to authenticate the user.",
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"dockercloud_node_cluster": resourceDockercloudNodeCluster(),
			"dockercloud_service":      resourceDockercloudService(),
			"dockercloud_stack":        resourceDockercloudStack(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := Config{
		User:    d.Get("user").(string),
		ApiKey:  d.Get("apikey").(string),
		BaseUrl: d.Get("baseurl").(string),
	}
	return config, config.Load()
}
