package scaleway

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/scaleway/scaleway-cli/pkg/api"
)

const gb uint64 = 1000 * 1000 * 1000

func resourceScalewayVolume() *schema.Resource {
	return &schema.Resource{
		Create: resourceScalewayVolumeCreate,
		Read:   resourceScalewayVolumeRead,
		Update: resourceScalewayVolumeUpdate,
		Delete: resourceScalewayVolumeDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"size_in_gb": {
				Type:         schema.TypeInt,
				Required:     true,
				ValidateFunc: validateVolumeSize,
			},
			"type": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateVolumeType,
			},
			"server": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceScalewayVolumeCreate(d *schema.ResourceData, m interface{}) error {
	scaleway := m.(*Client).scaleway

	mu.Lock()
	defer mu.Unlock()

	size := uint64(d.Get("size_in_gb").(int)) * gb
	req := api.ScalewayVolumeDefinition{
		Name:         d.Get("name").(string),
		Size:         size,
		Type:         d.Get("type").(string),
		Organization: scaleway.Organization,
	}
	volumeID, err := scaleway.PostVolume(req)
	if err != nil {
		return fmt.Errorf("Error Creating volume: %q", err)
	}
	d.SetId(volumeID)
	return resourceScalewayVolumeRead(d, m)
}

func resourceScalewayVolumeRead(d *schema.ResourceData, m interface{}) error {
	scaleway := m.(*Client).scaleway
	volume, err := scaleway.GetVolume(d.Id())
	if err != nil {
		if serr, ok := err.(api.ScalewayAPIError); ok {
			log.Printf("[DEBUG] Error reading volume: %q\n", serr.APIMessage)

			if serr.StatusCode == 404 {
				d.SetId("")
				return nil
			}
		}

		return err
	}
	d.Set("name", volume.Name)
	d.Set("size_in_gb", uint64(volume.Size)/gb)
	d.Set("type", volume.VolumeType)
	d.Set("server", "")
	if volume.Server != nil {
		d.Set("server", volume.Server.Identifier)
	}
	return nil
}

func resourceScalewayVolumeUpdate(d *schema.ResourceData, m interface{}) error {
	scaleway := m.(*Client).scaleway

	mu.Lock()
	defer mu.Unlock()

	var req api.ScalewayVolumePutDefinition
	if d.HasChange("name") {
		req.Name = String(d.Get("name").(string))
	}

	if d.HasChange("size_in_gb") {
		size := uint64(d.Get("size_in_gb").(int)) * gb
		req.Size = &size
	}

	scaleway.PutVolume(d.Id(), req)
	return resourceScalewayVolumeRead(d, m)
}

func resourceScalewayVolumeDelete(d *schema.ResourceData, m interface{}) error {
	scaleway := m.(*Client).scaleway

	mu.Lock()
	defer mu.Unlock()

	err := scaleway.DeleteVolume(d.Id())
	if err != nil {
		if serr, ok := err.(api.ScalewayAPIError); ok {
			if serr.StatusCode == 404 {
				d.SetId("")
				return nil
			}
		}
		return err
	}
	d.SetId("")
	return nil
}
