package clevercloud

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceCleverCloudApplicationPhp() *schema.Resource {
	return resourceCleverCloudApplication(
		"php",
		[]string{"par", "mtl"},
		[]string{"git", "ftp"},
		[]string{"nano", "xs", "s", "m", "l", "xl"},
	)
}
