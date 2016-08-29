package dockerregistry

import (
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"github.com/meteor/docker-registry-client/registry"
)

func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"username": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Username to log in to Docker registry",
				DefaultFunc: schema.EnvDefaultFunc("DOCKERREGISTRY_USERNAME", ""),
			},
			"password": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Password to log in to Docker registry",
				DefaultFunc: schema.EnvDefaultFunc("DOCKERREGISTRY_PASSWORD", ""),
			},
			"registry": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "https://registry-1.docker.io",
				Description: "URL to Docker V2 registry",
			},
		},
		DataSourcesMap: map[string]*schema.Resource{
			"dockerregistry_image": dataSourceImage(),
		},
		ConfigureFunc: providerConfigure,
	}
}

func dataSourceImage() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"repository": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true, // No Update command; we mutate Id
				Description: "Name of the repository in the registry; eg, `mycompany/myproject` or `library/alpine`",
			},
			"tag": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true, // No Update command; we mutate Id
				Description: "Tag to search for",
			},
		},

		Read: func(d *schema.ResourceData, meta interface{}) error {
			id, err := ensureImageExists(d, meta)
			if err != nil {
				return err
			}
			d.SetId(id)
			return nil
		},
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	registryURL := d.Get("registry").(string)
	reg := &registry.Registry{
		URL: registryURL,
		Client: &http.Client{
			Transport: registry.WrapTransport(http.DefaultTransport, registryURL,
				d.Get("username").(string), d.Get("password").(string)),
		},
		Logf: registry.Quiet,
	}
	return reg, nil
}

func ensureImageExists(d *schema.ResourceData, meta interface{}) (string, error) {
	reg := meta.(*registry.Registry)
	repository := d.Get("repository").(string)
	tag := d.Get("tag").(string)

	serverTags, err := reg.Tags(repository)
	if err != nil {
		return "", fmt.Errorf("Error looking up tags for %s: %s", repository, err)
	}

	if !stringInSlice(tag, serverTags) {
		return "", fmt.Errorf("Docker image %s:%s not found in registry", repository, tag)
	}

	return fmt.Sprintf("%s:%s", repository, tag), nil
}

// stringInSlice returns true if the string is an element of the slice.
//
// (It's great that Go makes it hard to ignore that this operation is O(n)!)
func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
