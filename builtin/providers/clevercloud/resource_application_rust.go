package clevercloud

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceCleverCloudApplicationRust() *schema.Resource {
	return resourceCleverCloudApplication(
		"rust",
		[]string{"par", "mtl"},
		[]string{"git"},
		[]string{"pico", "nano", "xs", "s", "m", "l", "xl"},
	)
}
