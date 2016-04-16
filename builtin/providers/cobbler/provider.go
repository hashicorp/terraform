package main

import (
	"net/http"

	cobbler "github.com/ContainerSolutions/cobblerclient"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"url": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Cobbler URL",
			},

			"username": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "The username for accessing Cobbler.",
			},

			"password": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "The password for accessing Cobbler.",
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"cobbler_system": resourceCobblerSystem(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := cobbler.ClientConfig{
		Url:      d.Get("url").(string),
		Username: d.Get("username").(string),
		Password: d.Get("password").(string),
	}

	client := cobbler.NewClient(http.DefaultClient, config)
	return &client, nil
}
