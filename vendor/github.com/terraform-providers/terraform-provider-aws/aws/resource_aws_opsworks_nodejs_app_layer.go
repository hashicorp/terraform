package aws

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsOpsworksNodejsAppLayer() *schema.Resource {
	layerType := &opsworksLayerType{
		TypeName:         "nodejs-app",
		DefaultLayerName: "Node.js App Server",

		Attributes: map[string]*opsworksLayerTypeAttribute{
			"nodejs_version": {
				AttrName: "NodejsVersion",
				Type:     schema.TypeString,
				Default:  "0.10.38",
			},
		},
	}

	return layerType.SchemaResource()
}
