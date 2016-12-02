package clevercloud

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceCleverCloudApplicationNode() *schema.Resource {
	return resourceCleverCloudApplication(
		"node",
		[]string{"par", "mtl"},
		[]string{"git"},
		[]string{"pico", "nano", "xs", "s", "m", "l", "xl"},
	)
}
