package aws

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsOpsworksHaproxyLayer() *schema.Resource {
	layerType := &opsworksLayerType{
		TypeName:         "lb",
		DefaultLayerName: "HAProxy",

		Attributes: map[string]*opsworksLayerTypeAttribute{
			"stats_enabled": {
				AttrName: "EnableHaproxyStats",
				Type:     schema.TypeBool,
				Default:  true,
			},
			"stats_url": {
				AttrName: "HaproxyStatsUrl",
				Type:     schema.TypeString,
				Default:  "/haproxy?stats",
			},
			"stats_user": {
				AttrName: "HaproxyStatsUser",
				Type:     schema.TypeString,
				Default:  "opsworks",
			},
			"stats_password": {
				AttrName:  "HaproxyStatsPassword",
				Type:      schema.TypeString,
				WriteOnly: true,
				Required:  true,
			},
			"healthcheck_url": {
				AttrName: "HaproxyHealthCheckUrl",
				Type:     schema.TypeString,
				Default:  "/",
			},
			"healthcheck_method": {
				AttrName: "HaproxyHealthCheckMethod",
				Type:     schema.TypeString,
				Default:  "OPTIONS",
			},
		},
	}

	return layerType.SchemaResource()
}
