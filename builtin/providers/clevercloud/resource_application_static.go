package clevercloud

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceCleverCloudApplicationStatic() *schema.Resource {
	return resourceCleverCloudApplication(
		"static",
		[]string{"par", "mtl"},
		[]string{"git"},
		[]string{"m", "l"},
	)
}
