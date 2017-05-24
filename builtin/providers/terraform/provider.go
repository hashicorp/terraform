package terraform

import (
	"github.com/r3labs/terraform/helper/schema"
	"github.com/r3labs/terraform/terraform"
)

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		ResourcesMap: map[string]*schema.Resource{
			"terraform_remote_state": schema.DataSourceResourceShim(
				"terraform_remote_state",
				dataSourceRemoteState(),
			),
		},
		DataSourcesMap: map[string]*schema.Resource{
			"terraform_remote_state": dataSourceRemoteState(),
		},
	}
}
