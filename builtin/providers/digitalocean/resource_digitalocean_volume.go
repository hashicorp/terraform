package digitalocean

import (
	"context"
	"fmt"
	"log"

	"github.com/digitalocean/godo"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceDigitalOceanVolume() *schema.Resource {
	return &schema.Resource{
		Create: resourceDigitalOceanVolumeCreate,
		Read:   resourceDigitalOceanVolumeRead,
		Delete: resourceDigitalOceanVolumeDelete,
		Importer: &schema.ResourceImporter{
			State: resourceDigitalOceanVolumeImport,
		},

		Schema: map[string]*schema.Schema{
			"region": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"droplet_ids": {
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeInt},
				Computed: true,
			},

			"size": {
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true, // Update-ability Coming Soon ™
			},

			"description": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true, // Update-ability Coming Soon ™
			},
		},
	}
}

func resourceDigitalOceanVolumeCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*godo.Client)

	opts := &godo.VolumeCreateRequest{
		Region:        d.Get("region").(string),
		Name:          d.Get("name").(string),
		Description:   d.Get("description").(string),
		SizeGigaBytes: int64(d.Get("size").(int)),
	}

	log.Printf("[DEBUG] Volume create configuration: %#v", opts)
	volume, _, err := client.Storage.CreateVolume(context.Background(), opts)
	if err != nil {
		return fmt.Errorf("Error creating Volume: %s", err)
	}

	d.SetId(volume.ID)
	log.Printf("[INFO] Volume name: %s", volume.Name)

	return resourceDigitalOceanVolumeRead(d, meta)
}

func resourceDigitalOceanVolumeRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*godo.Client)

	volume, resp, err := client.Storage.GetVolume(context.Background(), d.Id())
	if err != nil {
		// If the volume is somehow already destroyed, mark as
		// successfully gone
		if resp.StatusCode == 404 {
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error retrieving volume: %s", err)
	}

	d.Set("id", volume.ID)

	dids := make([]interface{}, 0, len(volume.DropletIDs))
	for _, did := range volume.DropletIDs {
		dids = append(dids, did)
	}
	d.Set("droplet_ids", schema.NewSet(
		func(dropletID interface{}) int { return dropletID.(int) },
		dids,
	))

	return nil
}

func resourceDigitalOceanVolumeDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*godo.Client)

	log.Printf("[INFO] Deleting volume: %s", d.Id())
	_, err := client.Storage.DeleteVolume(context.Background(), d.Id())
	if err != nil {
		return fmt.Errorf("Error deleting volume: %s", err)
	}

	d.SetId("")
	return nil
}

func resourceDigitalOceanVolumeImport(rs *schema.ResourceData, v interface{}) ([]*schema.ResourceData, error) {
	client := v.(*godo.Client)
	volume, _, err := client.Storage.GetVolume(context.Background(), rs.Id())
	if err != nil {
		return nil, err
	}

	rs.Set("id", volume.ID)
	rs.Set("name", volume.Name)
	rs.Set("region", volume.Region.Slug)
	rs.Set("description", volume.Description)
	rs.Set("size", int(volume.SizeGigaBytes))

	dids := make([]interface{}, 0, len(volume.DropletIDs))
	for _, did := range volume.DropletIDs {
		dids = append(dids, did)
	}
	rs.Set("droplet_ids", schema.NewSet(
		func(dropletID interface{}) int { return dropletID.(int) },
		dids,
	))

	return []*schema.ResourceData{rs}, nil
}
