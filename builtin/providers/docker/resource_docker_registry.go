package docker

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceDockerRegistry() *schema.Resource {
	return &schema.Resource{
		Create: resourceDockerRegistryRead,
		Delete: resourceDockerRegistryDelete,
		Read:   resourceDockerRegistryRead,
		Update: resourceDockerRegistryRead,

		Schema: map[string]*schema.Schema{
			"settings_file": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"auth"},
			},

			"auth": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"server_address": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"username": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"password": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"email": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},

			"configurations": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
		},
	}
}
