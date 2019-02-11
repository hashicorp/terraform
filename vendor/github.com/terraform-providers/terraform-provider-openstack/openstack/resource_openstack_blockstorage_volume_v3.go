package openstack

import (
	"fmt"
	"log"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/extensions/volumeactions"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/volumes"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/volumeattach"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceBlockStorageVolumeV3() *schema.Resource {
	return &schema.Resource{
		Create: resourceBlockStorageVolumeV3Create,
		Read:   resourceBlockStorageVolumeV3Read,
		Update: resourceBlockStorageVolumeV3Update,
		Delete: resourceBlockStorageVolumeV3Delete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"region": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"size": {
				Type:     schema.TypeInt,
				Required: true,
			},

			"enable_online_resize": {
				Type:     schema.TypeBool,
				Optional: true,
			},

			"name": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},

			"description": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},

			"availability_zone": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"metadata": {
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: false,
				Computed: true,
			},

			"snapshot_id": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"source_vol_id": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"image_id": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"volume_type": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"consistency_group_id": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"source_replica": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"multiattach": {
				Type:     schema.TypeBool,
				Optional: true,
			},

			"attachment": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"instance_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"device": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
				Set: blockStorageVolumeV3AttachmentHash,
			},
		},
	}
}

func resourceBlockStorageVolumeV3Create(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	blockStorageClient, err := config.blockStorageV3Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack block storage client: %s", err)
	}

	metadata := d.Get("metadata").(map[string]interface{})
	createOpts := &volumes.CreateOpts{
		AvailabilityZone:   d.Get("availability_zone").(string),
		ConsistencyGroupID: d.Get("consistency_group_id").(string),
		Description:        d.Get("description").(string),
		ImageID:            d.Get("image_id").(string),
		Metadata:           expandToMapStringString(metadata),
		Name:               d.Get("name").(string),
		Size:               d.Get("size").(int),
		SnapshotID:         d.Get("snapshot_id").(string),
		SourceReplica:      d.Get("source_replica").(string),
		SourceVolID:        d.Get("source_vol_id").(string),
		VolumeType:         d.Get("volume_type").(string),
		Multiattach:        d.Get("multiattach").(bool),
	}

	log.Printf("[DEBUG] openstack_blockstorage_volume_v3 create options: %#v", createOpts)

	v, err := volumes.Create(blockStorageClient, createOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error creating openstack_blockstorage_volume_v3: %s", err)
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"downloading", "creating"},
		Target:     []string{"available"},
		Refresh:    blockStorageVolumeV3StateRefreshFunc(blockStorageClient, v.ID),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf(
			"Error waiting for openstack_blockstorage_volume_v3 %s to become ready: %s", v.ID, err)
	}

	// Store the ID now
	d.SetId(v.ID)

	return resourceBlockStorageVolumeV3Read(d, meta)
}

func resourceBlockStorageVolumeV3Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	blockStorageClient, err := config.blockStorageV3Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack block storage client: %s", err)
	}

	v, err := volumes.Get(blockStorageClient, d.Id()).Extract()
	if err != nil {
		return CheckDeleted(d, err, "Error retrieving openstack_blockstorage_volume_v3")
	}

	log.Printf("[DEBUG] Retrieved openstack_blockstorage_volume_v3 %s: %#v", d.Id(), v)

	d.Set("size", v.Size)
	d.Set("description", v.Description)
	d.Set("availability_zone", v.AvailabilityZone)
	d.Set("name", v.Name)
	d.Set("snapshot_id", v.SnapshotID)
	d.Set("source_vol_id", v.SourceVolID)
	d.Set("volume_type", v.VolumeType)
	d.Set("metadata", v.Metadata)
	d.Set("region", GetRegion(d, config))

	attachments := flattenBlockStorageVolumeV3Attachments(v.Attachments)
	log.Printf("[DEBUG] openstack_blockstorage_volume_v3 %s attachments: %#v", d.Id(), attachments)
	if err := d.Set("attachment", attachments); err != nil {
		log.Printf(
			"[DEBUG] unable to set openstack_blockstorage_volume_v3 %s attachments: %s", d.Id(), err)
	}

	return nil
}

func resourceBlockStorageVolumeV3Update(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	blockStorageClient, err := config.blockStorageV3Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack block storage client: %s", err)
	}

	name := d.Get("name").(string)
	description := d.Get("description").(string)
	updateOpts := volumes.UpdateOpts{
		Name:        &name,
		Description: &description,
	}

	if d.HasChange("metadata") {
		metadata := d.Get("metadata").(map[string]interface{})
		updateOpts.Metadata = expandToMapStringString(metadata)
	}

	var v *volumes.Volume
	if d.HasChange("size") {
		v, err = volumes.Get(blockStorageClient, d.Id()).Extract()
		if err != nil {
			return fmt.Errorf("Error extending openstack_blockstorage_volume_v3 %s: %s", d.Id(), err)
		}

		if v.Status == "in-use" {
			if v, ok := d.Get("enable_online_resize").(bool); ok && !v {
				return fmt.Errorf(
					`Error extending openstack_blockstorage_volume_v3 %s,
					volume is attached to the instance and
					resizing online is disabled,
					see enable_online_resize option`, d.Id())
			}

			blockStorageClient.Microversion = "3.42"
		}

		extendOpts := volumeactions.ExtendSizeOpts{
			NewSize: d.Get("size").(int),
		}

		err = volumeactions.ExtendSize(blockStorageClient, d.Id(), extendOpts).ExtractErr()
		if err != nil {
			return fmt.Errorf("Error extending openstack_blockstorage_volume_v3 %s size: %s", d.Id(), err)
		}

		stateConf := &resource.StateChangeConf{
			Pending:    []string{"extending"},
			Target:     []string{"available", "in-use"},
			Refresh:    blockStorageVolumeV3StateRefreshFunc(blockStorageClient, d.Id()),
			Timeout:    d.Timeout(schema.TimeoutCreate),
			Delay:      10 * time.Second,
			MinTimeout: 3 * time.Second,
		}

		_, err := stateConf.WaitForState()
		if err != nil {
			return fmt.Errorf(
				"Error waiting for openstack_blockstorage_volume_v3 %s to become ready: %s", d.Id(), err)
		}
	}

	_, err = volumes.Update(blockStorageClient, d.Id(), updateOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error updating openstack_blockstorage_volume_v3 %s: %s", d.Id(), err)
	}

	return resourceBlockStorageVolumeV3Read(d, meta)
}

func resourceBlockStorageVolumeV3Delete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	blockStorageClient, err := config.blockStorageV3Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack block storage client: %s", err)
	}

	v, err := volumes.Get(blockStorageClient, d.Id()).Extract()
	if err != nil {
		return CheckDeleted(d, err, "Error retrieving openstack_blockstorage_volume_v3")
	}

	// make sure this volume is detached from all instances before deleting
	if len(v.Attachments) > 0 {
		computeClient, err := config.computeV2Client(GetRegion(d, config))
		if err != nil {
			return fmt.Errorf("Error creating OpenStack compute client: %s", err)
		}

		for _, volumeAttachment := range v.Attachments {
			log.Printf("[DEBUG] openstack_blockstorage_volume_v3 %s attachment: %#v", d.Id(), volumeAttachment)

			serverID := volumeAttachment.ServerID
			attachmentID := volumeAttachment.ID
			if err := volumeattach.Delete(computeClient, serverID, attachmentID).ExtractErr(); err != nil {
				// It's possible the volume was already detached by
				// openstack_compute_volume_attach_v2, so consider
				// a 404 acceptable and continue.
				if _, ok := err.(gophercloud.ErrDefault404); ok {
					continue
				}

				// A 409 is also acceptable because there's another
				// concurrent action happening.
				if errCode, ok := err.(gophercloud.ErrUnexpectedResponseCode); ok {
					if errCode.Actual == 409 {
						continue
					}
				}

				return fmt.Errorf(
					"Error detaching openstack_blockstorage_volume_v3 %s from %s: %s", d.Id(), serverID, err)
			}
		}

		stateConf := &resource.StateChangeConf{
			Pending:    []string{"in-use", "attaching", "detaching"},
			Target:     []string{"available", "deleted"},
			Refresh:    blockStorageVolumeV3StateRefreshFunc(blockStorageClient, d.Id()),
			Timeout:    10 * time.Minute,
			Delay:      10 * time.Second,
			MinTimeout: 3 * time.Second,
		}

		_, err = stateConf.WaitForState()
		if err != nil {
			return fmt.Errorf(
				"Error waiting for openstack_blockstorage_volume_v3 %s to become available: %s", d.Id(), err)
		}
	}

	// It's possible that this volume was used as a boot device and is currently
	// in a "deleting" state from when the instance was terminated.
	// If this is true, just move on. It'll eventually delete.
	if v.Status != "deleting" {
		if err := volumes.Delete(blockStorageClient, d.Id(), nil).ExtractErr(); err != nil {
			return CheckDeleted(d, err, "Error deleting openstack_blockstorage_volume_v3")
		}
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"deleting", "downloading", "available"},
		Target:     []string{"deleted"},
		Refresh:    blockStorageVolumeV3StateRefreshFunc(blockStorageClient, d.Id()),
		Timeout:    d.Timeout(schema.TimeoutDelete),
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for openstack_blockstorage_volume_v3 %s to delete: %s", d.Id(), err)
	}

	return nil
}
