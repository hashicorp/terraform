package docker

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceDockerImage() *schema.Resource {
	return &schema.Resource{
		Create: resourceDockerImageCreate,
		Read:   resourceDockerImageRead,
		Update: resourceDockerImageUpdate,
		Delete: resourceDockerImageDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"keep_updated": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},

			"latest": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"keep_locally": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
		},
	}
}
