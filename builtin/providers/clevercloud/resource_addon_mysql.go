package clevercloud

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceCleverCloudAddonMySQL() *schema.Resource {
	return resourceCleverCloudAddon(
		"mysql-addon",
		[]string{"dev", "s", "m", "ml", "l"},
		[]string{"eu", "us"},
	)
}
