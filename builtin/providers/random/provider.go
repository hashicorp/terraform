package random

import (
	"github.com/r3labs/terraform/helper/schema"
	"github.com/r3labs/terraform/terraform"
)

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{},

		ResourcesMap: map[string]*schema.Resource{
			"random_id":      resourceId(),
			"random_shuffle": resourceShuffle(),
			"random_pet":     resourcePet(),
		},
	}
}
