package scaleway

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/scaleway/scaleway-cli/pkg/api"
)

func resourceScalewayVolume() *schema.Resource {
	return &schema.Resource{
		Create: resourceScalewayVolumeCreate,
		Read:   resourceScalewayVolumeRead,
		Update: resourceScalewayVolumeUpdate,
		Delete: resourceScalewayVolumeDelete,
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"size": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(int)
					if value < 1000000000 || value > 150000000000 {
						errors = append(errors, fmt.Errorf("%q be more than 1000000000 and less than 150000000000", k))
					}
					return
				},
			},
			"type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)
					if value != "l_ssd" {
						errors = append(errors, fmt.Errorf("%q must be l_ssd", k))
					}
					return
				},
			},
			"server": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceScalewayVolumeCreate(d *schema.ResourceData, m interface{}) error {
	scaleway := m.(*Client).scaleway
	volumeID, err := scaleway.PostVolume(api.ScalewayVolumeDefinition{
		Name:         d.Get("name").(string),
		Size:         uint64(d.Get("size").(int)),
		Type:         d.Get("type").(string),
		Organization: scaleway.Organization,
	})
	if err != nil {
		serr := err.(api.ScalewayAPIError)

		return fmt.Errorf("Error Creating volumeReason: %s. %#v", serr.APIMessage, serr)
	}
	d.SetId(volumeID)
	return resourceScalewayVolumeRead(d, m)
}

func resourceScalewayVolumeRead(d *schema.ResourceData, m interface{}) error {
	scaleway := m.(*Client).scaleway
	volume, err := scaleway.GetVolume(d.Id())
	if err != nil {
		return err
	}
	d.Set("name", volume.Name)
	d.Set("size", volume.Size)
	d.Set("type", volume.VolumeType)
	d.Set("server", "")
	if volume.Server != nil {
		d.Set("server", volume.Server.Identifier)
	}
	return nil
}

func resourceScalewayVolumeUpdate(d *schema.ResourceData, m interface{}) error {
	scaleway := m.(*Client).scaleway

	var def api.ScalewayVolumePutDefinition
	if d.HasChange("name") {
		def.Name = String(d.Get("name").(string))
	}
	if d.HasChange("size") {
		size := uint64(d.Get("size").(int))
		def.Size = &size
	}
	scaleway.PutVolume(d.Id(), def)
	return resourceScalewayVolumeRead(d, m)
}

func resourceScalewayVolumeDelete(d *schema.ResourceData, m interface{}) error {
	scaleway := m.(*Client).scaleway
	err := scaleway.DeleteVolume(d.Id())
	if err != nil {
		return err
	}
	d.SetId("")
	return nil
}
