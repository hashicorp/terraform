package clevercloud

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceCleverCloudAddonPostgreSQL() *schema.Resource {
	return resourceCleverCloudAddon(
		"postgresql-addon",
		[]string{"dev", "s", "lm", "l", "xl"},
		[]string{"eu", "us"},
	)
}
