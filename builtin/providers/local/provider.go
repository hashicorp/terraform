package local

import (
	"github.com/r3labs/terraform/helper/schema"
	"github.com/r3labs/terraform/terraform"
)

func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{},
		ResourcesMap: map[string]*schema.Resource{
			"local_file": resourceLocalFile(),
		},
	}
}
