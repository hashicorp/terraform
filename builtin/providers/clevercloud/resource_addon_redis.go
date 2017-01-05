package clevercloud

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceCleverCloudAddonRedis() *schema.Resource {
	return resourceCleverCloudAddon(
		"redis-addon",
		[]string{"l", "xl", "xxl"},
		[]string{"eu"},
	)
}
