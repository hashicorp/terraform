package openstack

import (
	"fmt"
	"log"
	"time"

	"github.com/gophercloud/gophercloud/openstack/blockstorage/extensions/volumeactions"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/volumes"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceBlockStorageVolumeAttachV3() *schema.Resource {
	return &schema.Resource{
		Create: resourceBlockStorageVolumeAttachV3Create,
		Read:   resourceBlockStorageVolumeAttachV3Read,
		Delete: resourceBlockStorageVolumeAttachV3Delete,

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

			"volume_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"host_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"device": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"attach_mode": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				ValidateFunc: validation.StringInSlice([]string{
					"ro", "rw",
				}, false),
			},

			"initiator": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"ip_address": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"multipath": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},

			"os_type": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"platform": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"wwpn": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"wwnn": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			// Volume attachment information
			"data": {
				Type:      schema.TypeMap,
				Computed:  true,
				Sensitive: true,
			},

			"driver_volume_type": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"mount_point_base": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceBlockStorageVolumeAttachV3Create(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	client, err := config.blockStorageV3Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack block storage client: %s", err)
	}

	// initialize the connection
	volumeId := d.Get("volume_id").(string)
	connOpts := &volumeactions.InitializeConnectionOpts{}
	if v, ok := d.GetOk("host_name"); ok {
		connOpts.Host = v.(string)
	}

	if v, ok := d.GetOk("multipath"); ok {
		multipath := v.(bool)
		connOpts.Multipath = &multipath
	}

	if v, ok := d.GetOk("ip_address"); ok {
		connOpts.IP = v.(string)
	}

	if v, ok := d.GetOk("initiator"); ok {
		connOpts.Initiator = v.(string)
	}

	if v, ok := d.GetOk("os_type"); ok {
		connOpts.OSType = v.(string)
	}

	if v, ok := d.GetOk("platform"); ok {
		connOpts.Platform = v.(string)
	}

	if v, ok := d.GetOk("wwnns"); ok {
		connOpts.Wwnns = v.(string)
	}

	if v, ok := d.GetOk("wwpns"); ok {
		var wwpns []string
		for _, i := range v.([]string) {
			wwpns = append(wwpns, i)
		}

		connOpts.Wwpns = wwpns
	}

	connInfo, err := volumeactions.InitializeConnection(client, volumeId, connOpts).Extract()
	if err != nil {
		return fmt.Errorf(
			"Unable to initialize connection for openstack_blockstorage_volume_attach_v3: %s", err)
	}

	// Only uncomment this when debugging since connInfo contains sensitive information.
	// log.Printf("[DEBUG] Volume Connection for %s: %#v", volumeId, connInfo)

	// Because this information is only returned upon creation,
	// it must be set in Create.
	if v, ok := connInfo["data"]; ok {
		data := make(map[string]string)
		for key, value := range v.(map[string]interface{}) {
			if v, ok := value.(string); ok {
				data[key] = v
			}
		}

		d.Set("data", data)
	}

	if v, ok := connInfo["driver_volume_type"]; ok {
		d.Set("driver_volume_type", v)
	}

	if v, ok := connInfo["mount_point_base"]; ok {
		d.Set("mount_point_base", v)
	}

	// Once the connection has been made, tell Cinder to mark the volume as attached.
	attachMode, err := expandBlockStorageV3AttachMode(d.Get("attach_mode").(string))
	if err != nil {
		return nil
	}

	attachOpts := &volumeactions.AttachOpts{
		HostName:   d.Get("host_name").(string),
		MountPoint: d.Get("device").(string),
		Mode:       attachMode,
	}

	log.Printf("[DEBUG] openstack_blockstorage_volume_attach_v3 attach options: %#v", attachOpts)

	if err := volumeactions.Attach(client, volumeId, attachOpts).ExtractErr(); err != nil {
		return fmt.Errorf(
			"Error attaching openstack_blockstorage_volume_attach_v3 for volume %s: %s", volumeId, err)
	}

	// Wait for the volume to become available.
	log.Printf(
		"[DEBUG] Waiting for openstack_blockstorage_volume_attach_v3 volume %s to become available", volumeId)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"available", "attaching"},
		Target:     []string{"in-use"},
		Refresh:    blockStorageVolumeV3StateRefreshFunc(client, volumeId),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf(
			"Error waiting for openstack_blockstorage_volume_attach_v3 volume %s to become in-use: %s", volumeId, err)
	}

	// Once the volume has been marked as attached,
	// retrieve a fresh copy of it with all information now available.
	volume, err := volumes.Get(client, volumeId).Extract()
	if err != nil {
		return fmt.Errorf(
			"Unable to retrieve openstack_blockstorage_volume_attach_v3 volume %s: %s", volumeId, err)
	}

	// Search for the attachmentId
	var attachmentId string
	hostName := d.Get("host_name").(string)
	for _, attachment := range volume.Attachments {
		if hostName != "" && hostName == attachment.HostName {
			attachmentId = attachment.AttachmentID
		}
	}

	if attachmentId == "" {
		return fmt.Errorf(
			"Unable to determine attachment ID for openstack_blockstorage_volume_attach_v3 volume %s.", volumeId)
	}

	// The ID must be a combination of the volume and attachment ID
	// since a volume ID is required to retrieve an attachment ID.
	id := fmt.Sprintf("%s/%s", volumeId, attachmentId)
	d.SetId(id)

	return resourceBlockStorageVolumeAttachV3Read(d, meta)
}

func resourceBlockStorageVolumeAttachV3Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	client, err := config.blockStorageV3Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack block storage client: %s", err)
	}

	volumeId, attachmentId, err := blockStorageVolumeAttachV3ParseID(d.Id())
	if err != nil {
		return err
	}

	volume, err := volumes.Get(client, volumeId).Extract()
	if err != nil {
		return fmt.Errorf(
			"Unable to retrieve openstack_blockstorage_volume_attach_v3 volume %s: %s", volumeId, err)
	}

	log.Printf("[DEBUG] Retrieved openstack_blockstorage_volume_attach_v3 volume %s: %#v", volumeId, volume)

	var attachment volumes.Attachment
	for _, v := range volume.Attachments {
		if attachmentId == v.AttachmentID {
			attachment = v
		}
	}

	log.Printf(
		"[DEBUG] Retrieved openstack_blockstorage_volume_attach_v3 attachment %s: %#v", d.Id(), attachment)

	return nil
}

func resourceBlockStorageVolumeAttachV3Delete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	client, err := config.blockStorageV3Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack block storage client: %s", err)
	}

	volumeId, attachmentId, err := blockStorageVolumeAttachV3ParseID(d.Id())

	// Terminate the connection
	termOpts := &volumeactions.TerminateConnectionOpts{}
	if v, ok := d.GetOk("host_name"); ok {
		termOpts.Host = v.(string)
	}

	if v, ok := d.GetOk("multipath"); ok {
		multipath := v.(bool)
		termOpts.Multipath = &multipath
	}

	if v, ok := d.GetOk("ip_address"); ok {
		termOpts.IP = v.(string)
	}

	if v, ok := d.GetOk("initiator"); ok {
		termOpts.Initiator = v.(string)
	}

	if v, ok := d.GetOk("os_type"); ok {
		termOpts.OSType = v.(string)
	}

	if v, ok := d.GetOk("platform"); ok {
		termOpts.Platform = v.(string)
	}

	if v, ok := d.GetOk("wwnns"); ok {
		termOpts.Wwnns = v.(string)
	}

	if v, ok := d.GetOk("wwpns"); ok {
		var wwpns []string
		for _, i := range v.([]string) {
			wwpns = append(wwpns, i)
		}

		termOpts.Wwpns = wwpns
	}

	err = volumeactions.TerminateConnection(client, volumeId, termOpts).ExtractErr()
	if err != nil {
		return fmt.Errorf(
			"Error terminating openstack_blockstorage_volume_attach_v3 connection %s: %s", d.Id(), err)
	}

	// Detach the volume
	detachOpts := volumeactions.DetachOpts{
		AttachmentID: attachmentId,
	}

	log.Printf(
		"[DEBUG] openstack_blockstorage_volume_attach_v3 detachment options %s: %#v", d.Id(), detachOpts)

	if err := volumeactions.Detach(client, volumeId, detachOpts).ExtractErr(); err != nil {
		return err
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"in-use", "attaching", "detaching"},
		Target:     []string{"available"},
		Refresh:    blockStorageVolumeV3StateRefreshFunc(client, volumeId),
		Timeout:    d.Timeout(schema.TimeoutDelete),
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf(
			"Error waiting for openstack_blockstorage_volume_attach_v3 volume %s to become available: %s", volumeId, err)
	}

	return nil
}
