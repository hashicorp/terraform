package docker

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"host": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("DOCKER_HOST", "unix:/run/docker.sock"),
				Description: "The Docker daemon address",
			},

			"cert_path": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("DOCKER_CERT_PATH", nil),
				Description: "Path to directory with Docker TLS config",
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
		Host:     d.Get("host").(string),
		CertPath: d.Get("cert_path").(string),
	}

	return config.NewClient()
}
