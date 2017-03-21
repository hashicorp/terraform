package ns1

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"apikey": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("NS1_APIKEY", nil),
				Description: descriptions["api_key"],
			},
			"endpoint": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("NS1_ENDPOINT", nil),
				Description: descriptions["endpoint"],
			},
			"ignore_ssl": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("NS1_IGNORE_SSL", nil),
				Description: descriptions["ignore_ssl"],
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"ns1_zone":          zoneResource(),
			"ns1_record":        recordResource(),
			"ns1_datasource":    dataSourceResource(),
			"ns1_datafeed":      dataFeedResource(),
			"ns1_monitoringjob": monitoringJobResource(),
			"ns1_notifylist":    notifyListResource(),
			"ns1_user":          userResource(),
			"ns1_apikey":        apikeyResource(),
			"ns1_team":          teamResource(),
		},
		ConfigureFunc: ns1Configure,
	}
}

func ns1Configure(d *schema.ResourceData) (interface{}, error) {
	config := Config{
		Key: d.Get("apikey").(string),
	}

	if v, ok := d.GetOk("endpoint"); ok {
		config.Endpoint = v.(string)
	}
	if v, ok := d.GetOk("ignore_ssl"); ok {
		config.IgnoreSSL = v.(bool)
	}

	return config.Client()
}

var descriptions map[string]string

func init() {
	descriptions = map[string]string{
		"api_key": "The ns1 API key, this is required",
	}
}
