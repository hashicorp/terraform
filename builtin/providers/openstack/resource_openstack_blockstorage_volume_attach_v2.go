package openstack

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gophercloud/gophercloud/openstack/blockstorage/extensions/volumeactions"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v2/volumes"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceBlockStorageVolumeAttachV2() *schema.Resource {
	return &schema.Resource{
		Create: resourceBlockStorageVolumeAttachV2Create,
		Read:   resourceBlockStorageVolumeAttachV2Read,
		Delete: resourceBlockStorageVolumeAttachV2Delete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"region": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				DefaultFunc: schema.EnvDefaultFunc("OS_REGION_NAME", ""),
			},

			"volume_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"instance_id": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"host_name"},
			},

			"host_name": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"instance_id"},
			},

			"device": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"attach_mode": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)
					if value != "ro" && value != "rw" {
						errors = append(errors, fmt.Errorf(
							"Only 'ro' and 'rw' are supported values for 'attach_mode'"))
					}
					return
				},
			},
		},
	}
}

func resourceBlockStorageVolumeAttachV2Create(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	client, err := config.blockStorageV2Client(GetRegion(d))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack block storage client: %s", err)
	}

	// Check if either instance_id or host_name was set.
	instanceId := d.Get("instance_id").(string)
	hostName := d.Get("host_name").(string)
	if instanceId == "" && hostName == "" {
		return fmt.Errorf("One of 'instance_id' or 'host_name' must be set.")
	}

	volumeId := d.Get("volume_id").(string)

	attachMode, err := blockStorageVolumeAttachV2AttachMode(d.Get("attach_mode").(string))
	if err != nil {
		return nil
	}

	attachOpts := &volumeactions.AttachOpts{
		InstanceUUID: d.Get("instance_id").(string),
		HostName:     d.Get("host_name").(string),
		MountPoint:   d.Get("device").(string),
		Mode:         attachMode,
	}

	log.Printf("[DEBUG] Attachment Options: %#v", attachOpts)

	if err := volumeactions.Attach(client, volumeId, attachOpts).ExtractErr(); err != nil {
		return err
	}

	// Wait for the volume to become available.
	log.Printf("[DEBUG] Waiting for volume (%s) to become available", volumeId)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"available", "attaching"},
		Target:     []string{"in-use"},
		Refresh:    VolumeV2StateRefreshFunc(client, volumeId),
		Timeout:    10 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for volume (%s) to become ready: %s", volumeId, err)
	}

	volume, err := volumes.Get(client, volumeId).Extract()
	if err != nil {
		return err
	}

	var attachmentId string
	for _, attachment := range volume.Attachments {
		if instanceId != "" && instanceId == attachment.ServerID {
			attachmentId = attachment.AttachmentID
		}

		if hostName != "" && hostName == attachment.HostName {
			attachmentId = attachment.AttachmentID
		}
	}

	if attachmentId == "" {
		return fmt.Errorf("Unable to determine attachment ID.")
	}

	// The ID must be a combination of the volume and attachment ID
	// in order to import attachments.
	id := fmt.Sprintf("%s/%s", volumeId, attachmentId)
	d.SetId(id)

	return resourceBlockStorageVolumeAttachV2Read(d, meta)
}

func resourceBlockStorageVolumeAttachV2Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	client, err := config.blockStorageV2Client(GetRegion(d))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack block storage client: %s", err)
	}

	volumeId, attachmentId, err := blockStorageVolumeAttachV2ParseId(d.Id())
	if err != nil {
		return err
	}

	volume, err := volumes.Get(client, volumeId).Extract()
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Retrieved volume %s: %#v", d.Id(), volume)

	var attachment volumes.Attachment
	for _, v := range volume.Attachments {
		if attachmentId == v.AttachmentID {
			attachment = v
		}
	}

	log.Printf("[DEBUG] Retrieved volume attachment: %#v", attachment)

	d.Set("volume_id", volumeId)
	d.Set("attachment_id", attachmentId)
	d.Set("device", attachment.Device)
	d.Set("instance_id", attachment.ServerID)
	d.Set("host_name", attachment.HostName)
	d.Set("region", GetRegion(d))

	return nil
}

func resourceBlockStorageVolumeAttachV2Delete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	client, err := config.blockStorageV2Client(GetRegion(d))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack block storage client: %s", err)
	}

	volumeId, attachmentId, err := blockStorageVolumeAttachV2ParseId(d.Id())
	if err != nil {
		return err
	}

	detachOpts := volumeactions.DetachOpts{
		AttachmentID: attachmentId,
	}

	log.Printf("[DEBUG] Detachment Options: %#v", detachOpts)

	if err := volumeactions.Detach(client, volumeId, detachOpts).ExtractErr(); err != nil {
		return err
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"in-use", "attaching", "detaching"},
		Target:     []string{"available"},
		Refresh:    VolumeV2StateRefreshFunc(client, volumeId),
		Timeout:    10 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for volume (%s) to become available: %s", volumeId, err)
	}

	return nil
}

func blockStorageVolumeAttachV2AttachMode(v string) (volumeactions.AttachMode, error) {
	var attachMode volumeactions.AttachMode
	var attachError error
	switch v {
	case "":
		attachMode = ""
	case "ro":
		attachMode = volumeactions.ReadOnly
	case "rw":
		attachMode = volumeactions.ReadWrite
	default:
		attachError = fmt.Errorf("Invalid attach_mode specified")
	}

	return attachMode, attachError
}

func blockStorageVolumeAttachV2ParseId(id string) (string, string, error) {
	parts := strings.Split(id, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("Unable to determine attachment ID")
	}

	return parts[0], parts[1], nil
}
