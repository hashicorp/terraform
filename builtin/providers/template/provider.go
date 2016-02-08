package template

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		ResourcesMap: map[string]*schema.Resource{
			"template_file":             resourceFile(),
			"template_cloudinit_config": resourceCloudinitConfig(),
		},
	}
}
