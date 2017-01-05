package cloudfoundry

import (
	"fmt"

	"github.com/hashicorp/terraform/builtin/providers/cf/cfapi"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceInfo() *schema.Resource {

	return &schema.Resource{

		Read: dataSourceInfoRead,

		Schema: map[string]*schema.Schema{

			"api_version": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"auth_endpoint": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"uaa_endpoint": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"routing_endpoint": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"logging_endpoint": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"doppler_endpoint": &schema.Schema{
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
	d.Set("api_version", info.APIVersion)
	d.Set("auth_endpoint", info.AuthorizationEndpoint)
	d.Set("uaa_endpoint", info.TokenEndpoint)
	d.Set("routing_endpoint", info.RoutingAPIEndpoint)
	d.Set("logging_endpoint", info.LoggregatorEndpoint)
	d.Set("doppler_endpoint", info.DopplerEndpoint)

	d.SetId("info")
	return nil
}
