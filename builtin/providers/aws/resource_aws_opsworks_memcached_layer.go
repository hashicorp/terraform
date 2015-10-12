package aws

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsOpsworksMemcachedLayer() *schema.Resource {
	layerType := &opsworksLayerType{
		TypeName:         "memcached",
		DefaultLayerName: "Memcached",

		Attributes: map[string]*opsworksLayerTypeAttribute{
			"allocated_memory": &opsworksLayerTypeAttribute{
				AttrName: "MemcachedMemory",
				Type:     schema.TypeInt,
				Default:  512,
			},
		},
	}

	return layerType.SchemaResource()
}
