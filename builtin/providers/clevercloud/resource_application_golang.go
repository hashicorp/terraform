package clevercloud

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceCleverCloudApplicationGolang() *schema.Resource {
	return resourceCleverCloudApplication(
		"go",
		[]string{"par", "mtl"},
		[]string{"git"},
		[]string{"pico", "nano", "xs", "s", "m", "l", "xl"},
	)
}
