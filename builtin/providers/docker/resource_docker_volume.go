package docker

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceDockerVolume() *schema.Resource {
	return &schema.Resource{
		Create: resourceDockerVolumeCreate,
		Read:   resourceDockerVolumeRead,
		Delete: resourceDockerVolumeDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"driver": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"driver_opts": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},
			"mountpoint": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}
