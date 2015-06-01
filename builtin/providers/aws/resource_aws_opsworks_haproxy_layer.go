package aws

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsOpsworksHaproxyLayer() *schema.Resource {
	layerType := &opsworksLayerType{
		TypeName:         "lb",
		DefaultLayerName: "HAProxy",

		Attributes: map[string]*opsworksLayerTypeAttribute{
			"stats_enabled": &opsworksLayerTypeAttribute{
				AttrName: "EnableHaproxyStats",
				Type:     schema.TypeBool,
				Default:  true,
			},
			"stats_url": &opsworksLayerTypeAttribute{
				AttrName: "HaproxyStatsUrl",
				Type:     schema.TypeString,
				Default:  "/haproxy?stats",
			},
			"stats_user": &opsworksLayerTypeAttribute{
				AttrName: "HaproxyStatsUser",
				Type:     schema.TypeString,
				Default:  "opsworks",
			},
			"stats_password": &opsworksLayerTypeAttribute{
				AttrName:  "HaproxyStatsPassword",
				Type:      schema.TypeString,
				WriteOnly: true,
				Required:  true,
			},
			"healthcheck_url": &opsworksLayerTypeAttribute{
				AttrName: "HaproxyHealthCheckUrl",
				Type:     schema.TypeString,
				Default:  "/",
			},
			"healthcheck_method": &opsworksLayerTypeAttribute{
				AttrName: "HaproxyHealthCheckMethod",
				Type:     schema.TypeString,
				Default:  "OPTIONS",
			},
		},
	}

	return layerType.SchemaResource()
}
