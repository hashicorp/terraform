package docker

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"docker_host": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("DOCKER_HOST", "unix:/run/docker.sock"),
				Description: "The Docker daemon endpoint",
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"docker_container": resourceDockerContainer(),
			"docker_image":     resourceDockerImage(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := Config{
		DockerHost: d.Get("docker_host").(string),
	}

	return &config, nil
}
