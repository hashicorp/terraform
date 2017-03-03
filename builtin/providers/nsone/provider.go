package nsone

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
				DefaultFunc: schema.EnvDefaultFunc("NSONE_APIKEY", nil),
				Description: descriptions["api_key"],
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"nsone_zone":          zoneResource(),
			"nsone_record":        recordResource(),
			"nsone_datasource":    dataSourceResource(),
			"nsone_datafeed":      dataFeedResource(),
			"nsone_monitoringjob": monitoringJobResource(),
			"nsone_user":          userResource(),
			"nsone_apikey":        apikeyResource(),
			"nsone_team":          teamResource(),
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
		"api_key": "The nsone API key, this is required",
	}
}
