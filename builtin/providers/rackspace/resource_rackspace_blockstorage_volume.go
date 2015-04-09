package rackspace

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/rackspace/gophercloud"
	osVolumes "github.com/rackspace/gophercloud/openstack/blockstorage/v1/volumes"
	rsVolumes "github.com/rackspace/gophercloud/rackspace/blockstorage/v1/volumes"
)

func resourceBlockStorageVolume() *schema.Resource {
	return &schema.Resource{
		Create: resourceBlockStorageVolumeCreate,
		Read:   resourceBlockStorageVolumeRead,
		Update: resourceBlockStorageVolumeUpdate,
		Delete: resourceBlockStorageVolumeDelete,

		Schema: map[string]*schema.Schema{
			"region": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				DefaultFunc: envDefaultFunc("RS_REGION_NAME"),
			},
			"size": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},
			"metadata": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},
			"snapshot_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"source_vol_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"image_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"volume_type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceBlockStorageVolumeCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	blockStorageClient, err := config.blockStorageClient(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating Rackspace block storage client: %s", err)
	}

	createOpts := &osVolumes.CreateOpts{
		Description: d.Get("description").(string),
		Name:        d.Get("name").(string),
		Size:        d.Get("size").(int),
		SnapshotID:  d.Get("snapshot_id").(string),
		SourceVolID: d.Get("source_vol_id").(string),
		ImageID:     d.Get("image_id").(string),
		VolumeType:  d.Get("volume_type").(string),
		Metadata:    resourceVolumeMetadata(d),
	}

	log.Printf("[DEBUG] Create Options: %#v", createOpts)
	v, err := rsVolumes.Create(blockStorageClient, createOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error creating Rackspace volume: %s", err)
	}
	log.Printf("[INFO] Volume ID: %s", v.ID)

	// Store the ID now
	d.SetId(v.ID)

	// Wait for the volume to become available.
	log.Printf(
		"[DEBUG] Waiting for volume (%s) to become available",
		v.ID)

	stateConf := &resource.StateChangeConf{
		Target:     "available",
		Refresh:    VolumeStateRefreshFunc(blockStorageClient, v.ID),
		Timeout:    10 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf(
			"Error waiting for volume (%s) to become ready: %s",
			v.ID, err)
	}

	return resourceBlockStorageVolumeRead(d, meta)
}

func resourceBlockStorageVolumeRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	blockStorageClient, err := config.blockStorageClient(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating Rackspace block storage client: %s", err)
	}

	v, err := rsVolumes.Get(blockStorageClient, d.Id()).Extract()
	if err != nil {
		return CheckDeleted(d, err, "volume")
	}

	log.Printf("[DEBUG] Retreived volume %s: %+v", d.Id(), v)

	d.Set("size", v.Size)
	d.Set("description", v.Description)
	d.Set("name", v.Name)
	d.Set("snapshot_id", v.SnapshotID)
	d.Set("source_vol_id", v.SourceVolID)
	d.Set("volume_type", v.VolumeType)
	d.Set("metadata", v.Metadata)

	return nil

}

func resourceBlockStorageVolumeUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	blockStorageClient, err := config.blockStorageClient(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating Rackspace block storage client: %s", err)
	}

	updateOpts := rsVolumes.UpdateOpts{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
	}

	_, err = rsVolumes.Update(blockStorageClient, d.Id(), updateOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error updating Rackspace volume: %s", err)
	}

	return resourceBlockStorageVolumeRead(d, meta)
}

func resourceBlockStorageVolumeDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	blockStorageClient, err := config.blockStorageClient(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating Rackspace block storage client: %s", err)
	}

	err = rsVolumes.Delete(blockStorageClient, d.Id()).ExtractErr()
	if err != nil {
		return fmt.Errorf("Error deleting Rackspace volume: %s", err)
	}

	// Wait for the volume to delete before moving on.
	log.Printf("[DEBUG] Waiting for volume (%s) to delete", d.Id())

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"deleting", "available"},
		Target:     "deleted",
		Refresh:    VolumeStateRefreshFunc(blockStorageClient, d.Id()),
		Timeout:    10 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf(
			"Error waiting for volume (%s) to delete: %s",
			d.Id(), err)
	}

	d.SetId("")
	return nil
}

func resourceVolumeMetadata(d *schema.ResourceData) map[string]string {
	m := make(map[string]string)
	for key, val := range d.Get("metadata").(map[string]interface{}) {
		m[key] = val.(string)
	}
	return m
}

// VolumeStateRefreshFunc returns a resource.StateRefreshFunc that is used to watch
// an OpenStack volume.
func VolumeStateRefreshFunc(client *gophercloud.ServiceClient, volumeID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		v, err := rsVolumes.Get(client, volumeID).Extract()
		if err != nil {
			errCode, ok := err.(*gophercloud.UnexpectedResponseCodeError)
			if !ok {
				return nil, "", err
			}
			if errCode.Actual == 404 {
				return v, "deleted", nil
			}
			return nil, "", err
		}

		return v, v.Status, nil
	}
}
