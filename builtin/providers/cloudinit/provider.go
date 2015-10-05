package cloudinit

import (
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/terraform/helper/schema"
)

func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		ResourcesMap: map[string]*schema.Resource{
			"cloudinit_config": resource(),
		},
	}
}
