package ns1

import (
	"net/http"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"

	nsone "gopkg.in/ns1/ns1-go.v2/rest"
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
		},
		ResourcesMap: map[string]*schema.Resource{
			"ns1_zone":          zoneResource(),
			"ns1_record":        recordResource(),
			"ns1_datasource":    dataSourceResource(),
			"ns1_datafeed":      dataFeedResource(),
			"ns1_monitoringjob": monitoringJobResource(),
			"ns1_user":          userResource(),
			"ns1_apikey":        apikeyResource(),
			"ns1_team":          teamResource(),
		},
		ConfigureFunc: nsoneConfigure,
	}
}

func nsoneConfigure(d *schema.ResourceData) (interface{}, error) {
	httpClient := &http.Client{}
	n := nsone.NewClient(httpClient, nsone.SetAPIKey(d.Get("apikey").(string)))
	// FIXME: n.Debug()
	// FIXME: n.RateLimitStrategySleep()
	return n, nil
}

var descriptions map[string]string

func init() {
	descriptions = map[string]string{
		"api_key": "The ns1 API key, this is required",
	}
}
