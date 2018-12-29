package profitbricks

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/profitbricks/profitbricks-sdk-go"
	"log"
)

func resourceProfitBricksVolume() *schema.Resource {
	return &schema.Resource{
		Create: resourceProfitBricksVolumeCreate,
		Read:   resourceProfitBricksVolumeRead,
		Update: resourceProfitBricksVolumeUpdate,
		Delete: resourceProfitBricksVolumeDelete,
		Schema: map[string]*schema.Schema{
			"image_name": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"size": {
				Type:     schema.TypeInt,
				Required: true,
			},

			"disk_type": {
				Type:     schema.TypeString,
				Required: true,
			},
			"image_password": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"licence_type": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"ssh_key_path": {
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
			},
			"sshkey": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"bus": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"name": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"availability_zone": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"server_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"datacenter_id": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceProfitBricksVolumeCreate(d *schema.ResourceData, meta interface{}) error {
	var err error
	var ssh_keypath []interface{}
	dcId := d.Get("datacenter_id").(string)
	serverId := d.Get("server_id").(string)
	imagePassword := d.Get("image_password").(string)
	ssh_keypath = d.Get("ssh_key_path").([]interface{})
	image_name := d.Get("image_name").(string)

	if image_name != "" {
		if imagePassword == "" && len(ssh_keypath) == 0 {
			return fmt.Errorf("Either 'image_password' or 'sshkey' must be provided.")
		}
	}

	licenceType := d.Get("licence_type").(string)

	if image_name == "" && licenceType == "" {
		return fmt.Errorf("Either 'image_name', or 'licenceType' must be set.")
	}

	var publicKeys []string
	if len(ssh_keypath) != 0 {
		for _, path := range ssh_keypath {
			log.Printf("[DEBUG] Reading file %s", path)
			publicKey, err := readPublicKey(path.(string))
			if err != nil {
				return fmt.Errorf("Error fetching sshkey from file (%s) (%s)", path, err.Error())
			}
			publicKeys = append(publicKeys, publicKey)
		}
	}

	var image string
	if !IsValidUUID(image_name) {
		image = getImageId(d.Get("datacenter_id").(string), image_name, d.Get("disk_type").(string))
	} else {
		image = image_name
	}

	volume := profitbricks.Volume{
		Properties: profitbricks.VolumeProperties{
			Name:          d.Get("name").(string),
			Size:          d.Get("size").(int),
			Type:          d.Get("disk_type").(string),
			ImagePassword: imagePassword,
			Image:         image,
			Bus:           d.Get("bus").(string),
			LicenceType:   licenceType,
		},
	}

	if len(publicKeys) != 0 {
		volume.Properties.SshKeys = publicKeys

	} else {
		volume.Properties.SshKeys = nil
	}

	if _, ok := d.GetOk("availability_zone"); ok {
		raw := d.Get("availability_zone").(string)
		volume.Properties.AvailabilityZone = raw
	}

	volume = profitbricks.CreateVolume(dcId, volume)

	if volume.StatusCode > 299 {
		return fmt.Errorf("An error occured while creating a volume: %s", volume.Response)
	}

	err = waitTillProvisioned(meta, volume.Headers.Get("Location"))
	if err != nil {
		return err
	}
	volume = profitbricks.AttachVolume(dcId, serverId, volume.Id)
	if volume.StatusCode > 299 {
		return fmt.Errorf("An error occured while attaching a volume dcId: %s server_id: %s ID: %s Response: %s", dcId, serverId, volume.Id, volume.Response)
	}

	err = waitTillProvisioned(meta, volume.Headers.Get("Location"))
	if err != nil {
		return err
	}
	d.SetId(volume.Id)

	return resourceProfitBricksVolumeRead(d, meta)
}

func resourceProfitBricksVolumeRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	profitbricks.SetAuth(config.Username, config.Password)

	dcId := d.Get("datacenter_id").(string)

	volume := profitbricks.GetVolume(dcId, d.Id())

	if volume.StatusCode > 299 {
		if volume.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error occured while fetching a volume ID %s %s", d.Id(), volume.Response)
	}

	if volume.StatusCode > 299 {
		return fmt.Errorf("An error occured while fetching a volume ID %s %s", d.Id(), volume.Response)

	}

	d.Set("name", volume.Properties.Name)
	d.Set("disk_type", volume.Properties.Type)
	d.Set("size", volume.Properties.Size)
	d.Set("bus", volume.Properties.Bus)
	d.Set("image_name", volume.Properties.Image)

	return nil
}

func resourceProfitBricksVolumeUpdate(d *schema.ResourceData, meta interface{}) error {
	properties := profitbricks.VolumeProperties{}
	dcId := d.Get("datacenter_id").(string)

	if d.HasChange("name") {
		_, newValue := d.GetChange("name")
		properties.Name = newValue.(string)
	}
	if d.HasChange("disk_type") {
		_, newValue := d.GetChange("disk_type")
		properties.Type = newValue.(string)
	}
	if d.HasChange("size") {
		_, newValue := d.GetChange("size")
		properties.Size = newValue.(int)
	}
	if d.HasChange("bus") {
		_, newValue := d.GetChange("bus")
		properties.Bus = newValue.(string)
	}
	if d.HasChange("availability_zone") {
		_, newValue := d.GetChange("availability_zone")
		properties.AvailabilityZone = newValue.(string)
	}

	volume := profitbricks.PatchVolume(dcId, d.Id(), properties)
	err := waitTillProvisioned(meta, volume.Headers.Get("Location"))
	if err != nil {
		return err
	}
	if volume.StatusCode > 299 {
		return fmt.Errorf("An error occured while updating a volume ID %s %s", d.Id(), volume.Response)

	}
	err = resourceProfitBricksVolumeRead(d, meta)
	if err != nil {
		return err
	}
	d.SetId(d.Get("server_id").(string))
	err = resourceProfitBricksServerRead(d, meta)
	if err != nil {
		return err
	}

	d.SetId(volume.Id)
	return nil
}

func resourceProfitBricksVolumeDelete(d *schema.ResourceData, meta interface{}) error {
	dcId := d.Get("datacenter_id").(string)

	resp := profitbricks.DeleteVolume(dcId, d.Id())
	if resp.StatusCode > 299 {
		return fmt.Errorf("An error occured while deleting a volume ID %s %s", d.Id(), string(resp.Body))

	}
	err := waitTillProvisioned(meta, resp.Headers.Get("Location"))
	if err != nil {
		return err
	}
	d.SetId("")
	return nil
}
