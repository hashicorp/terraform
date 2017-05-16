package aws

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsOpsworksJavaAppLayer() *schema.Resource {
	layerType := &opsworksLayerType{
		TypeName:         "java-app",
		DefaultLayerName: "Java App Server",

		Attributes: map[string]*opsworksLayerTypeAttribute{
			"jvm_type": {
				AttrName: "Jvm",
				Type:     schema.TypeString,
				Default:  "openjdk",
			},
			"jvm_version": {
				AttrName: "JvmVersion",
				Type:     schema.TypeString,
				Default:  "7",
			},
			"jvm_options": {
				AttrName: "JvmOptions",
				Type:     schema.TypeString,
				Default:  "",
			},
			"app_server": {
				AttrName: "JavaAppServer",
				Type:     schema.TypeString,
				Default:  "tomcat",
			},
			"app_server_version": {
				AttrName: "JavaAppServerVersion",
				Type:     schema.TypeString,
				Default:  "7",
			},
		},
	}

	return layerType.SchemaResource()
}
