package template

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		DataSourcesMap: map[string]*schema.Resource{
			"template_file":             dataSourceFile(),
			"template_cloudinit_config": dataSourceCloudinitConfig(),
		},
		ResourcesMap: map[string]*schema.Resource{
			"template_file": schema.DataSourceResourceShim(
				"template_file",
				dataSourceFile(),
			),
			"template_cloudinit_config": schema.DataSourceResourceShim(
				"template_cloudinit_config",
				dataSourceCloudinitConfig(),
			),
			"template_dir": resourceDir(),
		},
	}
}
