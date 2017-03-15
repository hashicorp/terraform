package puppetdb

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"url": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("PUPPETDB_URL", "http://localhost:8080"),
				Description: descriptions["url"],
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"puppetdb_node": resourcePuppetDBNode(),
		},

		ConfigureFunc: providerConfigure,
	}
}

var descriptions map[string]string

func init() {
	descriptions = map[string]string{
		"url": "The URL to the PuppetDB",
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	url := d.Get("url").(string)

	client := PuppetDBClient{
		URL: url,
	}
	return &client, nil
}
