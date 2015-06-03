package aws

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsOpsworksJavaAppLayer() *schema.Resource {
	layerType := &opsworksLayerType{
		TypeName:         "java-app",
		DefaultLayerName: "Java App Server",

		Attributes: map[string]*opsworksLayerTypeAttribute{
			"jvm_type": &opsworksLayerTypeAttribute{
				AttrName: "Jvm",
				Type:     schema.TypeString,
				Default:  "openjdk",
			},
			"jvm_version": &opsworksLayerTypeAttribute{
				AttrName: "JvmVersion",
				Type:     schema.TypeString,
				Default:  "7",
			},
			"jvm_options": &opsworksLayerTypeAttribute{
				AttrName: "JvmOptions",
				Type:     schema.TypeString,
				Default:  "",
			},
			"app_server": &opsworksLayerTypeAttribute{
				AttrName: "JavaAppServer",
				Type:     schema.TypeString,
				Default:  "tomcat",
			},
			"app_server_version": &opsworksLayerTypeAttribute{
				AttrName: "JavaAppServerVersion",
				Type:     schema.TypeString,
				Default:  "7",
			},
		},
	}

	return layerType.SchemaResource()
}
