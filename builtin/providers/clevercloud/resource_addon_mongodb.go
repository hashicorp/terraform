package clevercloud

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceCleverCloudAddonMongoDB() *schema.Resource {
	return resourceCleverCloudAddon(
		"mongodb-addon",
		[]string{"s", "sm", "m"},
		[]string{"eu", "us"},
	)
}
