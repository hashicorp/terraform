package docker

import (
	"fmt"

	dc "github.com/fsouza/go-dockerclient"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceDockerVolumeCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*dc.Client)

	createOpts := dc.CreateVolumeOptions{
		Name: d.Get("name").(string),
	}
	if v, ok := d.GetOk("driver"); ok {
		createOpts.Driver = v.(string)
	}
	if v, ok := d.GetOk("driver_opts"); ok {
		createOpts.DriverOpts = mapTypeMapValsToString(v.(map[string]interface{}))
	}

	var err error
	var retVolume *dc.Volume
	if retVolume, err = client.CreateVolume(createOpts); err != nil {
		return fmt.Errorf("Unable to create volume: %s", err)
	}
	if retVolume == nil {
		return fmt.Errorf("Returned volume is nil")
	}

	d.SetId(retVolume.Name)

	return nil
}

func resourceDockerVolumeRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*dc.Client)

	var err error
	var retVolume *dc.Volume
	if retVolume, err = client.InspectVolume(d.Id()); err != nil {
		return fmt.Errorf("Unable to inspect volume: %s", err)
	}
	if retVolume == nil {
		d.SetId("")
		return nil
	}

	d.Set("driver", retVolume.Driver)
	d.Set("mountpoint", retVolume.Mountpoint)

	return nil
}

func resourceDockerVolumeDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*dc.Client)

	if err := client.RemoveVolume(d.Id()); err != nil {
		return fmt.Errorf("Error deleting volume %s: %s", d.Id(), err)
	}

	d.SetId("")
	return nil
}
