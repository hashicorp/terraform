package azure

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func SchemaSubscription(subscriptionIDOptional bool) map[string]*schema.Schema {
	s := map[string]*schema.Schema{
		"subscription_id": {
			Type:     schema.TypeString,
			Optional: subscriptionIDOptional,
			Computed: true,
		},

		"display_name": {
			Type:     schema.TypeString,
			Computed: true,
		},

		"state": {
			Type:     schema.TypeString,
			Computed: true,
		},

		"location_placement_id": {
			Type:     schema.TypeString,
			Computed: true,
		},

		"quota_id": {
			Type:     schema.TypeString,
			Computed: true,
		},

		"spending_limit": {
			Type:     schema.TypeString,
			Computed: true,
		},
	}

	return s
}
