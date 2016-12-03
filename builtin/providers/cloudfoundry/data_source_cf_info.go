package cloudfoundry

import (
	"fmt"

	"github.com/hashicorp/terraform/builtin/providers/cloudfoundry/cfapi"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceInfo() *schema.Resource {

	return &schema.Resource{

		Read: dataSourceInfoRead,

		Schema: map[string]*schema.Schema{

			"api-version": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"auth-endpoint": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"uaa-endpoint": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"routing-endpoint": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"logging-endpoint": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"doppler-endpoint": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceInfoRead(d *schema.ResourceData, meta interface{}) error {

	session := meta.(*cfapi.Session)
	if session == nil {
		return fmt.Errorf("client is nil")
	}

	info := session.Info()
	d.Set("api-version", info.APIVersion)
	d.Set("auth-endpoint", info.AuthorizationEndpoint)
	d.Set("uaa-endpoint", info.TokenEndpoint)
	d.Set("routing-endpoint", info.RoutingAPIEndpoint)
	d.Set("logging-endpoint", info.LoggregatorEndpoint)
	d.Set("doppler-endpoint", info.DopplerEndpoint)

	d.SetId("info")
	return nil
}
