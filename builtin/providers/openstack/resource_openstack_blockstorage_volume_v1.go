package openstack

import (
	"bytes"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack/blockstorage/v1/volumes"
	"github.com/rackspace/gophercloud/openstack/compute/v2/extensions/volumeattach"
)

func resourceBlockStorageVolumeV1() *schema.Resource {
	return &schema.Resource{
		Create: resourceBlockStorageVolumeV1Create,
		Read:   resourceBlockStorageVolumeV1Read,
		Update: resourceBlockStorageVolumeV1Update,
		Delete: resourceBlockStorageVolumeV1Delete,

		Schema: map[string]*schema.Schema{
			"region": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				DefaultFunc: envDefaultFuncAllowMissing("OS_REGION_NAME"),
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
			"availability_zone": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},
			"metadata": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: false,
				Computed: true,
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
				Computed: true,
			},
			"attachment": &schema.Schema{
				Type:     schema.TypeSet,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						"instance_id": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						"device": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
				Set: resourceVolumeAttachmentHash,
			},
		},
	}
}

func resourceBlockStorageVolumeV1Create(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	blockStorageClient, err := config.blockStorageV1Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack block storage client: %s", err)
	}

	createOpts := &volumes.CreateOpts{
		Description:  d.Get("description").(string),
		Availability: d.Get("availability_zone").(string),
		Name:         d.Get("name").(string),
		Size:         d.Get("size").(int),
		SnapshotID:   d.Get("snapshot_id").(string),
		SourceVolID:  d.Get("source_vol_id").(string),
		ImageID:      d.Get("image_id").(string),
		VolumeType:   d.Get("volume_type").(string),
		Metadata:     resourceContainerMetadataV2(d),
	}

	log.Printf("[DEBUG] Create Options: %#v", createOpts)
	v, err := volumes.Create(blockStorageClient, createOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error creating OpenStack volume: %s", err)
	}
	log.Printf("[INFO] Volume ID: %s", v.ID)

	// Store the ID now
	d.SetId(v.ID)

	// Wait for the volume to become available.
	log.Printf(
		"[DEBUG] Waiting for volume (%s) to become available",
		v.ID)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"downloading", "creating"},
		Target:     "available",
		Refresh:    VolumeV1StateRefreshFunc(blockStorageClient, v.ID),
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

	return resourceBlockStorageVolumeV1Read(d, meta)
}

func resourceBlockStorageVolumeV1Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	blockStorageClient, err := config.blockStorageV1Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack block storage client: %s", err)
	}

	v, err := volumes.Get(blockStorageClient, d.Id()).Extract()
	if err != nil {
		return CheckDeleted(d, err, "volume")
	}

	log.Printf("[DEBUG] Retreived volume %s: %+v", d.Id(), v)

	d.Set("size", v.Size)
	d.Set("description", v.Description)
	d.Set("availability_zone", v.AvailabilityZone)
	d.Set("name", v.Name)
	d.Set("snapshot_id", v.SnapshotID)
	d.Set("source_vol_id", v.SourceVolID)
	d.Set("volume_type", v.VolumeType)
	d.Set("metadata", v.Metadata)

	if len(v.Attachments) > 0 {
		attachments := make([]map[string]interface{}, len(v.Attachments))
		for i, attachment := range v.Attachments {
			attachments[i] = make(map[string]interface{})
			attachments[i]["id"] = attachment["id"]
			attachments[i]["instance_id"] = attachment["server_id"]
			attachments[i]["device"] = attachment["device"]
			log.Printf("[DEBUG] attachment: %v", attachment)
		}
		d.Set("attachment", attachments)
	}

	return nil
}

func resourceBlockStorageVolumeV1Update(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	blockStorageClient, err := config.blockStorageV1Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack block storage client: %s", err)
	}

	updateOpts := volumes.UpdateOpts{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
	}

	if d.HasChange("metadata") {
		updateOpts.Metadata = resourceVolumeMetadataV1(d)
	}

	_, err = volumes.Update(blockStorageClient, d.Id(), updateOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error updating OpenStack volume: %s", err)
	}

	return resourceBlockStorageVolumeV1Read(d, meta)
}

func resourceBlockStorageVolumeV1Delete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	blockStorageClient, err := config.blockStorageV1Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack block storage client: %s", err)
	}

	v, err := volumes.Get(blockStorageClient, d.Id()).Extract()
	if err != nil {
		return CheckDeleted(d, err, "volume")
	}

	// make sure this volume is detached from all instances before deleting
	if len(v.Attachments) > 0 {
		log.Printf("[DEBUG] detaching volumes")
		if computeClient, err := config.computeV2Client(d.Get("region").(string)); err != nil {
			return err
		} else {
			for _, volumeAttachment := range v.Attachments {
				log.Printf("[DEBUG] Attachment: %v", volumeAttachment)
				if err := volumeattach.Delete(computeClient, volumeAttachment["server_id"].(string), volumeAttachment["id"].(string)).ExtractErr(); err != nil {
					return err
				}
			}

			stateConf := &resource.StateChangeConf{
				Pending:    []string{"in-use", "attaching"},
				Target:     "available",
				Refresh:    VolumeV1StateRefreshFunc(blockStorageClient, d.Id()),
				Timeout:    10 * time.Minute,
				Delay:      10 * time.Second,
				MinTimeout: 3 * time.Second,
			}

			_, err = stateConf.WaitForState()
			if err != nil {
				return fmt.Errorf(
					"Error waiting for volume (%s) to become available: %s",
					d.Id(), err)
			}
		}
	}

	// It's possible that this volume was used as a boot device and is currently
	// in a "deleting" state from when the instance was terminated.
	// If this is true, just move on. It'll eventually delete.
	if v.Status != "deleting" {
		if err := volumes.Delete(blockStorageClient, d.Id()).ExtractErr(); err != nil {
			return CheckDeleted(d, err, "volume")
		}
	}

	// Wait for the volume to delete before moving on.
	log.Printf("[DEBUG] Waiting for volume (%s) to delete", d.Id())

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"deleting", "downloading", "available"},
		Target:     "deleted",
		Refresh:    VolumeV1StateRefreshFunc(blockStorageClient, d.Id()),
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

func resourceVolumeMetadataV1(d *schema.ResourceData) map[string]string {
	m := make(map[string]string)
	for key, val := range d.Get("metadata").(map[string]interface{}) {
		m[key] = val.(string)
	}
	return m
}

// VolumeV1StateRefreshFunc returns a resource.StateRefreshFunc that is used to watch
// an OpenStack volume.
func VolumeV1StateRefreshFunc(client *gophercloud.ServiceClient, volumeID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		v, err := volumes.Get(client, volumeID).Extract()
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

func resourceVolumeAttachmentHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	if m["instance_id"] != nil {
		buf.WriteString(fmt.Sprintf("%s-", m["instance_id"].(string)))
	}
	return hashcode.String(buf.String())
}
