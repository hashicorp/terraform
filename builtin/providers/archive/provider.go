package archive

import (
	"github.com/r3labs/terraform/helper/schema"
	"github.com/r3labs/terraform/terraform"
)

func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		DataSourcesMap: map[string]*schema.Resource{
			"archive_file": dataSourceFile(),
		},
		ResourcesMap: map[string]*schema.Resource{
			"archive_file": schema.DataSourceResourceShim(
				"archive_file",
				dataSourceFile(),
			),
		},
	}
}
