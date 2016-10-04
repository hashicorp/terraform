package akamai

import (
	"github.com/akamai-open/AkamaiOPEN-edgegrid-golang"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"access_token": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("AKAMAI_ACCESS_TOKEN", nil),
				Description: "",
			},

			"client_secret": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("AKAMAI_CLIENT_SECRET", nil),
				Description: "",
			},

			"client_token": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("AKAMAI_CLIENT_TOKEN", nil),
				Description: "",
			},

			"host": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("AKAMAI_HOST", nil),
				Description: "",
			},
		},

		DataSourcesMap: map[string]*schema.Resource{
		//	"akamai_contract": dataSourceAkamaiContract(),
		},

		ResourcesMap: map[string]*schema.Resource{
			//"akamai_edge_hostname": resourceEdgeHostname(),
			//"akamai_property": resourceProperty(),
			//"akamai_property_rule": resourcePropertyRule(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := &edgegrid.Config{
		AccessToken:  d.Get("access_token").(string),
		ClientSecret: d.Get("client_secret").(string),
		ClientToken:  d.Get("client_token").(string),
		Host:         d.Get("host").(string),
	}

	client := &Client{
		Config: config,
	}

	return client, nil
}
