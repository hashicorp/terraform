package grafana

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"

	gapi "github.com/apparentlymart/go-grafana-api"
)

func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"url": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("GRAFANA_URL", nil),
				Description: "URL of the root of the target Grafana server.",
			},
			"auth": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("GRAFANA_AUTH", nil),
				Description: "Credentials for accessing the Grafana API.",
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"grafana_dashboard":                   ResourceDashboard(),
			"grafana_dashboard_config":            ResourceDashboardConfig(),
			//"grafana_graph_panel_config":          ResourceGraphPanelConfig(),
			//"grafana_single_stat_panel_config":    ResourceSingleStatPanelConfig(),
			"grafana_text_panel_config":           ResourceTextPanelConfig(),
			"grafana_dashboard_list_panel_config": ResourceDashboardListPanelConfig(),
			"grafana_data_source":                 ResourceDataSource(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	return gapi.New(
		d.Get("auth").(string),
		d.Get("url").(string),
	)
}
