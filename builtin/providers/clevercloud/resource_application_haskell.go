package clevercloud

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceCleverCloudApplicationHaskell() *schema.Resource {
	return resourceCleverCloudApplication(
		"haskell",
		[]string{"par", "mtl"},
		[]string{"git"},
		[]string{"pico", "nano", "xs", "s", "m", "l", "xl"},
	)
}
