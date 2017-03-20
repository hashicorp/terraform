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
			"cert": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("PUPPETDB_CERT", ""),
				Description: descriptions["cert"],
			},
			"key": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("PUPPETDB_KEY", ""),
				Description: descriptions["key"],
			},
			"ca": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("PUPPETDB_CA", ""),
				Description: descriptions["ca"],
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
		"url":  "The URL to the PuppetDB",
		"cert": "The SSL certificate to connect to the PuppetDB",
		"key":  "The SSL private key to connect to the PuppetDB",
		"ca":   "The SSL CA certificate to connect to the PuppetDB",
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	url := d.Get("url").(string)
	cert := d.Get("cert").(string)
	key := d.Get("key").(string)
	ca := d.Get("ca").(string)

	client := PuppetDBClient{
		URL:  url,
		Cert: cert,
		Key:  key,
		CA:   ca,
	}
	return &client, nil
}
