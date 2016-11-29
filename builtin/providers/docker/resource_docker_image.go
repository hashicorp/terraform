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

			"latest": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"keep_locally": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},

			"pull_trigger": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}
