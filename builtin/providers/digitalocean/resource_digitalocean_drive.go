package digitalocean

import (
	"fmt"
	"log"
	"strconv"

	"github.com/digitalocean/godo"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceDigitalOceanDrive() *schema.Resource {
	return &schema.Resource{
		Create: resourceDigitalOceanDriveCreate,
		Read:   resourceDigitalOceanDriveRead,
		Update: resourceDigitalOceanDriveUpdate,
		Delete: resourceDigitalOceanDriveDelete,

		Schema: map[string]*schema.Schema{
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"description": &schema.Schema{
				Type:     schema.TypeString
			}

			"size": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			}
		},
	}
}

func resourceDigitalOceanDriveCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*godo.Client)

	// Build up our creation options
	opts := &godo.DriveCreateRequest{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
		Size:        d.Get("size").(string),
	}

	log.Printf("[DEBUG] Drive create configuration: %#v", opts)
	drive, _, err := client.Drives.Create(opts)
	if err != nil {
		return fmt.Errorf("Error creating Drive: %s", err)
	}

	d.SetId(strconv.Itoa(drive.ID))
	log.Printf("[INFO] Drive: %d", drive.ID)

	return resourceDigitalOceanDriveRead(d, meta)
}

func resourceDigitalOceanDriveRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*godo.Client)

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return fmt.Errorf("invalid Drive id: %v", err)
	}

	drive, resp, err := client.Drives.GetByID(id)
	if err != nil {
		// If the drive is somehow already destroyed, mark as
		// successfully gone
		if resp != nil && resp.StatusCode == 404 {
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error retrieving Drive: %s", err)
	}

	d.Set("name", drive.Name)
	d.Set("description", drive.Description)
	d.Set("size", drive.Size)

	return nil
}

func resourceDigitalOceanDriveUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*godo.Client)

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return fmt.Errorf("invalid Drive id: %v", err)
	}

	var newName string
	if v, ok := d.GetOk("name"); ok {
		newName = v.(string)
	}

	log.Printf("[DEBUG] Drive update name: %#v", newName)
	opts := &godo.DriveUpdateRequest{
		Name: newName,
	}
	_, _, err = client.Drives.UpdateByID(id, opts)
	if err != nil {
		return fmt.Errorf("Failed to update Drive: %s", err)
	}

	return resourceDigitalOceanDriveRead(d, meta)
}

func resourceDigitalOceanDriveDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*godo.Client)

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return fmt.Errorf("invalid Drive id: %v", err)
	}

	log.Printf("[INFO] Deleting Drive: %d", id)
	_, err = client.Drives.DeleteByID(id)
	if err != nil {
		return fmt.Errorf("Error deleting Drive: %s", err)
	}

	d.SetId("")
	return nil
}
