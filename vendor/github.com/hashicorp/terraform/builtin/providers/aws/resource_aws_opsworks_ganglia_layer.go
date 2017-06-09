package aws

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsOpsworksGangliaLayer() *schema.Resource {
	layerType := &opsworksLayerType{
		TypeName:         "monitoring-master",
		DefaultLayerName: "Ganglia",

		Attributes: map[string]*opsworksLayerTypeAttribute{
			"url": {
				AttrName: "GangliaUrl",
				Type:     schema.TypeString,
				Default:  "/ganglia",
			},
			"username": {
				AttrName: "GangliaUser",
				Type:     schema.TypeString,
				Default:  "opsworks",
			},
			"password": {
				AttrName:  "GangliaPassword",
				Type:      schema.TypeString,
				Required:  true,
				WriteOnly: true,
			},
		},
	}

	return layerType.SchemaResource()
}
