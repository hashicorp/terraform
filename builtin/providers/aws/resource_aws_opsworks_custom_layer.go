package aws

import (
	"github.com/r3labs/terraform/helper/schema"
)

func resourceAwsOpsworksCustomLayer() *schema.Resource {
	layerType := &opsworksLayerType{
		TypeName:        "custom",
		CustomShortName: true,

		// The "custom" layer type has no additional attributes
		Attributes: map[string]*opsworksLayerTypeAttribute{},
	}

	return layerType.SchemaResource()
}
