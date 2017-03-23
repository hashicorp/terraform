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

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
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
				Type:       schema.TypeString,
				Optional:   true,
				ForceNew:   true,
				Deprecated: "instance_id is no longer used in this resource",
			},

			"host_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"device": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
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

			"initiator": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"ip_address": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"multipath": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},

			"os_type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"platform": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"wwpn": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"wwnn": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			// Volume attachment information
			"data": &schema.Schema{
				Type:      schema.TypeMap,
				Computed:  true,
				Sensitive: true,
			},

			"driver_volume_type": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"mount_point_base": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
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
		return fmt.Errorf("Unable to create connection: %s", err)
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
	attachMode, err := blockStorageVolumeAttachV2AttachMode(d.Get("attach_mode").(string))
	if err != nil {
		return nil
	}

	attachOpts := &volumeactions.AttachOpts{
		HostName:   d.Get("host_name").(string),
		MountPoint: d.Get("device").(string),
		Mode:       attachMode,
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
		Timeout:    d.Timeout(schema.TimeoutCreate),
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for volume (%s) to become ready: %s", volumeId, err)
	}

	// Once the volume has been marked as attached,
	// retrieve a fresh copy of it with all information now available.
	volume, err := volumes.Get(client, volumeId).Extract()
	if err != nil {
		return err
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
		return fmt.Errorf("Unable to determine attachment ID.")
	}

	// The ID must be a combination of the volume and attachment ID
	// since a volume ID is required to retrieve an attachment ID.
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

	return nil
}

func resourceBlockStorageVolumeAttachV2Delete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	client, err := config.blockStorageV2Client(GetRegion(d))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack block storage client: %s", err)
	}

	volumeId, attachmentId, err := blockStorageVolumeAttachV2ParseId(d.Id())

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
		return fmt.Errorf("Error terminating volume connection %s: %s", volumeId, err)
	}

	// Detach the volume
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
		Timeout:    d.Timeout(schema.TimeoutDelete),
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
