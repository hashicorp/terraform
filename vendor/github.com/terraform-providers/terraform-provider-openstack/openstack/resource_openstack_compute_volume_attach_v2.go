package openstack

import (
	"fmt"
	"log"
	"time"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/volumeattach"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceComputeVolumeAttachV2() *schema.Resource {
	return &schema.Resource{
		Create: resourceComputeVolumeAttachV2Create,
		Read:   resourceComputeVolumeAttachV2Read,
		Delete: resourceComputeVolumeAttachV2Delete,
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

			"instance_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"volume_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"device": {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
			},

			"multiattach": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceComputeVolumeAttachV2Create(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	computeClient, err := config.computeV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack compute client: %s", err)
	}

	instanceId := d.Get("instance_id").(string)
	volumeId := d.Get("volume_id").(string)

	var device string
	if v, ok := d.GetOk("device"); ok {
		device = v.(string)
	}

	attachOpts := volumeattach.CreateOpts{
		Device:   device,
		VolumeID: volumeId,
	}

	log.Printf("[DEBUG] openstack_compute_volume_attach_v2 attach options %s: %#v", instanceId, attachOpts)

	if v := d.Get("multiattach").(bool); v {
		computeClient.Microversion = "2.60"
	}

	attachment, err := volumeattach.Create(computeClient, instanceId, attachOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error creating openstack_compute_volume_attach_v2 %s: %s", instanceId, err)
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"ATTACHING"},
		Target:     []string{"ATTACHED"},
		Refresh:    computeVolumeAttachV2AttachFunc(computeClient, instanceId, attachment.ID),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	if _, err = stateConf.WaitForState(); err != nil {
		return fmt.Errorf("Error attaching openstack_compute_volume_attach_v2 %s: %s", instanceId, err)
	}

	// Use the instance ID and attachment ID as the resource ID.
	// This is because an attachment cannot be retrieved just by its ID alone.
	id := fmt.Sprintf("%s/%s", instanceId, attachment.ID)

	d.SetId(id)

	return resourceComputeVolumeAttachV2Read(d, meta)
}

func resourceComputeVolumeAttachV2Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	computeClient, err := config.computeV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack compute client: %s", err)
	}

	instanceId, attachmentId, err := computeVolumeAttachV2ParseID(d.Id())
	if err != nil {
		return err
	}

	attachment, err := volumeattach.Get(computeClient, instanceId, attachmentId).Extract()
	if err != nil {
		return CheckDeleted(d, err, "Error retrieving openstack_compute_volume_attach_v2")
	}

	log.Printf("[DEBUG] Retrieved openstack_compute_volume_attach_v2 %s: %#v", d.Id(), attachment)

	d.Set("instance_id", attachment.ServerID)
	d.Set("volume_id", attachment.VolumeID)
	d.Set("device", attachment.Device)
	d.Set("region", GetRegion(d, config))

	return nil
}

func resourceComputeVolumeAttachV2Delete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	computeClient, err := config.computeV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack compute client: %s", err)
	}

	instanceId, attachmentId, err := computeVolumeAttachV2ParseID(d.Id())
	if err != nil {
		return err
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{""},
		Target:     []string{"DETACHED"},
		Refresh:    computeVolumeAttachV2DetachFunc(computeClient, instanceId, attachmentId),
		Timeout:    d.Timeout(schema.TimeoutDelete),
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	if _, err = stateConf.WaitForState(); err != nil {
		return CheckDeleted(d, err, "Error detaching openstack_compute_volume_attach_v2")
	}

	return nil
}
