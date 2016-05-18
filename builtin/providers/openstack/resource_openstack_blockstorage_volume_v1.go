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
				DefaultFunc: schema.EnvDefaultFunc("OS_REGION_NAME", ""),
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
				Optional: true,
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
	vol, err := volumes.Create(blockStorageClient, createOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error creating OpenStack volume: %s", err)
	}
	log.Printf("[INFO] Volume ID: %s", vol.ID)

	// Wait for the volume to become available.
	log.Printf("[DEBUG] Waiting for volume (%s) to become available", vol.ID)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"downloading", "creating"},
		Target:     []string{"available"},
		Refresh:    volumeV1StateRefreshFunc(blockStorageClient, vol.ID),
		Timeout:    10 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for volume (%s) to become ready: %s", vol.ID, err)
	}

	// Store the ID now
	d.SetId(vol.ID)

	// If attachments were specified, attach the volume to the instances.
	// In theory, this supports multi-attach, though at this time, it'll
	// most likely be a single attachment.
	if v, ok := d.GetOk("attachment"); ok {
		computeClient, err := config.computeV2Client(d.Get("region").(string))
		if err != nil {
			return fmt.Errorf("Error creating OpenStack compute client: %s", err)
		}

		attachments := v.(*schema.Set).List()
		for _, attachment := range attachments {
			attachmentMap := attachment.(map[string]interface{})
			attachmentInfo := make(map[string]interface{})

			if v, ok := attachmentMap["device"].(string); ok && v != "" {
				attachmentInfo["deviceName"] = v
			}

			if v, ok := attachmentMap["instance_id"].(string); ok && v != "" {
				attachmentInfo["instanceID"] = v
			}

			attachmentInfo["volumeID"] = vol.ID

			if _, err := attachVolumeToInstance(computeClient, blockStorageClient, attachmentInfo); err != nil {
				return fmt.Errorf("Error attaching volume %s to instance %s: %s", vol.ID, attachmentInfo["instanceID"], err)
			}
		}
	}

	return resourceBlockStorageVolumeV1Read(d, meta)
}

func resourceBlockStorageVolumeV1Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	blockStorageClient, err := config.blockStorageV1Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack block storage client: %s", err)
	}

	vol, err := volumes.Get(blockStorageClient, d.Id()).Extract()
	if err != nil {
		return CheckDeleted(d, err, "volume")
	}

	log.Printf("[DEBUG] Retreived volume %s: %+v", d.Id(), vol)

	d.Set("size", vol.Size)
	d.Set("description", vol.Description)
	d.Set("availability_zone", vol.AvailabilityZone)
	d.Set("name", vol.Name)
	d.Set("snapshot_id", vol.SnapshotID)
	d.Set("source_vol_id", vol.SourceVolID)
	d.Set("volume_type", vol.VolumeType)
	d.Set("metadata", vol.Metadata)

	if len(vol.Attachments) > 0 {
		attachments := make([]map[string]interface{}, len(vol.Attachments))
		for i, attachment := range vol.Attachments {
			attachments[i] = make(map[string]interface{})
			attachments[i]["id"] = attachment["id"]
			attachments[i]["device"] = attachment["device"]
			attachments[i]["instance_id"] = attachment["server_id"]
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

	if d.HasChange("attachment") {
		computeClient, err := config.computeV2Client(d.Get("region").(string))
		if err != nil {
			return fmt.Errorf("Error creating OpenStack compute client: %s", err)
		}

		o, n := d.GetChange("attachment")

		oldAttachments := o.(*schema.Set).List()
		for _, oldAttachment := range oldAttachments {
			oldAttachmentMap := make(map[string]interface{})
			attachmentInfo := make(map[string]interface{})

			if v, ok := oldAttachment.(map[string]interface{}); ok && v != nil {
				oldAttachmentMap = v
			}

			if v, ok := oldAttachmentMap["id"].(string); ok && v != "" {
				attachmentInfo["attachmentID"] = v
			}

			if v, ok := oldAttachmentMap["instance_id"].(string); ok && v != "" {
				attachmentInfo["instanceID"] = v
			}

			attachmentInfo["volumeID"] = d.Id()

			if attachmentInfo["attachmentID"] != nil && attachmentInfo["instanceID"] != nil {
				// for each old attachment, detach the volume
				if err := detachVolumeFromInstance(computeClient, blockStorageClient, attachmentInfo); err != nil {
					return CheckDeleted(d, err, fmt.Sprintf("Error detaching volume %s from instance %s", d.Id(), attachmentInfo["instanceID"]))
				}
			}
		}

		newAttachments := n.(*schema.Set).List()
		for _, newAttachment := range newAttachments {
			newAttachmentMap := make(map[string]interface{})
			attachmentInfo := make(map[string]interface{})

			if v, ok := newAttachment.(map[string]interface{}); ok && v != nil {
				newAttachmentMap = v
			}

			if v, ok := newAttachmentMap["id"].(string); ok && v != "" {
				attachmentInfo["attachmentID"] = v
			}

			if v, ok := newAttachmentMap["instance_id"].(string); ok && v != "" {
				attachmentInfo["instanceID"] = v
			}

			if v, ok := newAttachmentMap["device_name"].(string); ok && v != "" {
				attachmentInfo["deviceName"] = v
			}

			attachmentInfo["volumeID"] = d.Id()

			if attachmentInfo["instanceID"] != nil {
				// for each new attachment that isn't already attached, attach the volume
				if _, err := attachVolumeToInstance(computeClient, blockStorageClient, attachmentInfo); err != nil {
					return fmt.Errorf("Error attaching volume %s to instance %s: %s", d.Id(), attachmentInfo["instanceID"], err)
				}
			}
		}
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

		computeClient, err := config.computeV2Client(d.Get("region").(string))
		if err != nil {
			return err
		}

		for _, volumeAttachment := range v.Attachments {
			log.Printf("[DEBUG] Attachment: %v", volumeAttachment)

			attachmentInfo := make(map[string]interface{})
			if v, ok := volumeAttachment["id"].(string); ok && v != "" {
				attachmentInfo["attachmentID"] = v
			}

			if v, ok := volumeAttachment["server_id"].(string); ok && v != "" {
				attachmentInfo["instanceID"] = v
			}

			if v, ok := volumeAttachment["device_name"].(string); ok && v != "" {
				attachmentInfo["deviceName"] = v
			}

			attachmentInfo["volumeID"] = d.Id()

			if attachmentInfo["instanceID"] != nil {
				// for each new attachment that isn't already attached, attach the volume
				if err := detachVolumeFromInstance(computeClient, blockStorageClient, attachmentInfo); err != nil {
					return fmt.Errorf("Error detaching volume %s from instance %s: %s", d.Id(), attachmentInfo["instanceID"], err)
				}
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
		Target:     []string{"deleted"},
		Refresh:    volumeV1StateRefreshFunc(blockStorageClient, d.Id()),
		Timeout:    10 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for volume (%s) to delete: %s", d.Id(), err)
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

// volumeV1StateRefreshFunc returns a resource.StateRefreshFunc that is used to watch
// an OpenStack volume.
func volumeV1StateRefreshFunc(blockStorageClient *gophercloud.ServiceClient, volumeID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		v, err := volumes.Get(blockStorageClient, volumeID).Extract()
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

// attachVolumeToInstance attaches a single volume to a single instance.
func attachVolumeToInstance(computeClient, blockStorageClient *gophercloud.ServiceClient, attachmentInfo map[string]interface{}) (*volumeattach.VolumeAttachment, error) {
	var instanceID string
	if v, ok := attachmentInfo["instanceID"].(string); ok && v != "" {
		instanceID = v
	}

	var volumeID string
	if v, ok := attachmentInfo["volumeID"].(string); ok && v != "" {
		volumeID = v
	}

	var deviceName string
	if v, ok := attachmentInfo["deviceName"].(string); ok && v != "" {
		deviceName = v
	}

	vaOpts := &volumeattach.CreateOpts{
		VolumeID: volumeID,
		Device:   deviceName,
	}

	log.Printf("[DEBUG] Attempting to attach volume %s to instance %s", volumeID, instanceID)
	attachInfo, err := volumeattach.Create(computeClient, instanceID, vaOpts).Extract()
	if err != nil {
		return nil, err
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"attaching", "available"},
		Target:     []string{"in-use"},
		Refresh:    volumeV1StateRefreshFunc(blockStorageClient, volumeID),
		Timeout:    10 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return nil, fmt.Errorf("Error waiting for volume (%s) to become available: %s", volumeID, err)
	}

	return attachInfo, nil
}

// detachVolumeFromInstance detaches a single volume from a single instance.
func detachVolumeFromInstance(computeClient, blockStorageClient *gophercloud.ServiceClient, attachmentInfo map[string]interface{}) error {
	var instanceID string
	if v, ok := attachmentInfo["instanceID"].(string); ok && v != "" {
		instanceID = v
	}

	var attachmentID string
	if v, ok := attachmentInfo["attachmentID"].(string); ok && v != "" {
		attachmentID = v
	}

	var volumeID string
	if v, ok := attachmentInfo["volumeID"].(string); ok && v != "" {
		volumeID = v
	}

	log.Printf("[DEBUG] Attempting to detach volume %s from instance %s", volumeID, instanceID)
	err := volumeattach.Delete(computeClient, instanceID, attachmentID).ExtractErr()
	if err != nil {
		return err
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"in-use", "attaching", "detaching"},
		Target:     []string{"available"},
		Refresh:    volumeV1StateRefreshFunc(blockStorageClient, volumeID),
		Timeout:    10 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for volume (%s) to become available: %s", volumeID, err)
	}

	return nil
}

func resourceVolumeAttachmentHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	if m["instance_id"] != nil {
		buf.WriteString(fmt.Sprintf("%s-", m["instance_id"].(string)))
	}
	return hashcode.String(buf.String())
}
