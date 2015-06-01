package aws

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsOpsworksRailsAppLayer() *schema.Resource {
	layerType := &opsworksLayerType{
		TypeName:         "rails-app",
		DefaultLayerName: "Rails App Server",

		Attributes: map[string]*opsworksLayerTypeAttribute{
			"ruby_version": &opsworksLayerTypeAttribute{
				AttrName: "RubyVersion",
				Type:     schema.TypeString,
				Default:  "2.0.0",
			},
			"app_server": &opsworksLayerTypeAttribute{
				AttrName: "RailsStack",
				Type:     schema.TypeString,
				Default:  "apache_passenger",
			},
			"passenger_version": &opsworksLayerTypeAttribute{
				AttrName: "PassengerVersion",
				Type:     schema.TypeString,
				Default:  "4.0.46",
			},
			"rubygems_version": &opsworksLayerTypeAttribute{
				AttrName: "RubygemsVersion",
				Type:     schema.TypeString,
				Default:  "2.2.2",
			},
			"manage_bundler": &opsworksLayerTypeAttribute{
				AttrName: "ManageBundler",
				Type:     schema.TypeBool,
				Default:  true,
			},
			"bundler_version": &opsworksLayerTypeAttribute{
				AttrName: "BundlerVersion",
				Type:     schema.TypeString,
				Default:  "1.5.3",
			},
		},
	}

	return layerType.SchemaResource()
}
