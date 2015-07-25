package nsone

import (
	"github.com/bobtfish/go-nsone-api"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"apikey": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("NSONE_APIKEY", nil),
				Description: descriptions["api_key"],
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"nsone_zone":       zoneResource(),
			"nsone_record":     recordResource(),
			"nsone_datasource": dataSourceResource(),
		},
		ConfigureFunc: nsoneConfigure,
	}
}

func nsoneConfigure(d *schema.ResourceData) (interface{}, error) {
	n := nsone.New(d.Get("apikey").(string))
	return n, nil
}

var descriptions map[string]string

func init() {
	descriptions = map[string]string{
		"api_key": "The nsone API key, this is required",
	}
}
