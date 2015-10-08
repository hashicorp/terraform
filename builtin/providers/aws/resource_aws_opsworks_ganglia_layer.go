package aws

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsOpsworksGangliaLayer() *schema.Resource {
	layerType := &opsworksLayerType{
		TypeName:         "monitoring-master",
		DefaultLayerName: "Ganglia",

		Attributes: map[string]*opsworksLayerTypeAttribute{
			"url": &opsworksLayerTypeAttribute{
				AttrName: "GangliaUrl",
				Type:     schema.TypeString,
				Default:  "/ganglia",
			},
			"username": &opsworksLayerTypeAttribute{
				AttrName: "GangliaUser",
				Type:     schema.TypeString,
				Default:  "opsworks",
			},
			"password": &opsworksLayerTypeAttribute{
				AttrName:  "GangliaPassword",
				Type:      schema.TypeString,
				Required:  true,
				WriteOnly: true,
			},
		},
	}

	return layerType.SchemaResource()
}
