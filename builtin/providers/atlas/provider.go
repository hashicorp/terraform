package atlas

import (
	"github.com/hashicorp/atlas-go/v1"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

const (
	// defaultAtlasServer is the default endpoint for Atlas if
	// none is specified.
	defaultAtlasServer = "https://atlas.hashicorp.com"
)

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"token": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("ATLAS_TOKEN", nil),
				Description: descriptions["token"],
			},

			"address": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("ATLAS_ADDRESS", defaultAtlasServer),
				Description: descriptions["address"],
			},
		},

		DataSourcesMap: map[string]*schema.Resource{
			"atlas_artifact": dataSourceAtlasArtifact(),
		},

		ResourcesMap: map[string]*schema.Resource{
			"atlas_artifact": resourceArtifact(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	var err error
	client := atlas.DefaultClient()
	if v := d.Get("address").(string); v != "" {
		client, err = atlas.NewClient(v)
		if err != nil {
			return nil, err
		}
	}
	client.DefaultHeader.Set(terraform.VersionHeader, terraform.VersionString())
	client.Token = d.Get("token").(string)

	return client, nil
}

var descriptions map[string]string

func init() {
	descriptions = map[string]string{
		"address": "The address of the Atlas server. If blank, the public\n" +
			"server at atlas.hashicorp.com will be used.",

		"token": "The access token for reading artifacts. This is required\n" +
			"if reading private artifacts.",
	}
}
