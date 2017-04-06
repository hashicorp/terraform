package opc

import (
	"fmt"
	"strconv"

	"github.com/hashicorp/go-oracle-terraform/compute"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceOPCStorageVolumeSnapshot() *schema.Resource {
	return &schema.Resource{
		Create: resourceOPCStorageVolumeSnapshotCreate,
		Read:   resourceOPCStorageVolumeSnapshotRead,
		Delete: resourceOPCStorageVolumeSnapshotDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			// Required Attributes
			"volume_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			// Optional Attributes
			"description": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			// Optional, but also computed if unspecified
			"name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"parent_volume_bootable": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
				Default:  false,
			},

			"collocated": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
				ForceNew: true,
			},

			"tags": tagsForceNewSchema(),

			// Computed Attributes
			"account": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"machine_image_name": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"size": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"property": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"platform": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"snapshot_timestamp": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"snapshot_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"start_timestamp": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"status_detail": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"status_timestamp": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"uri": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceOPCStorageVolumeSnapshotCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*compute.Client).StorageVolumeSnapshots()

	// Get required attribute
	input := &compute.CreateStorageVolumeSnapshotInput{
		Volume: d.Get("volume_name").(string),
	}

	if v, ok := d.GetOk("description"); ok {
		input.Description = v.(string)
	}

	if v, ok := d.GetOk("name"); ok {
		input.Name = v.(string)
	}

	// Convert parent_volume_bootable to string
	bootable := d.Get("parent_volume_bootable").(bool)
	if bootable {
		input.ParentVolumeBootable = "true"
	}

	collocated := d.Get("collocated").(bool)
	if collocated {
		input.Property = compute.SnapshotPropertyCollocated
	}

	tags := getStringList(d, "tags")
	if len(tags) > 0 {
		input.Tags = tags
	}

	info, err := client.CreateStorageVolumeSnapshot(input)
	if err != nil {
		return fmt.Errorf("Error creating snapshot '%s': %v", input.Name, err)
	}

	d.SetId(info.Name)
	return resourceOPCStorageVolumeSnapshotRead(d, meta)
}

func resourceOPCStorageVolumeSnapshotRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*compute.Client).StorageVolumeSnapshots()

	name := d.Id()
	input := &compute.GetStorageVolumeSnapshotInput{
		Name: name,
	}

	result, err := client.GetStorageVolumeSnapshot(input)
	if err != nil {
		if compute.WasNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading storage volume snapshot '%s': %v", name, err)
	}

	d.Set("volume_name", result.Volume)
	d.Set("description", result.Description)
	d.Set("name", result.Name)
	d.Set("property", result.Property)
	d.Set("platform", result.Platform)
	d.Set("account", result.Account)
	d.Set("machine_image_name", result.MachineImageName)
	d.Set("size", result.Size)
	d.Set("snapshot_timestamp", result.SnapshotTimestamp)
	d.Set("snapshot_id", result.SnapshotID)
	d.Set("start_timestamp", result.StartTimestamp)
	d.Set("status", result.Status)
	d.Set("status_detail", result.StatusDetail)
	d.Set("status_timestamp", result.StatusTimestamp)
	d.Set("uri", result.URI)

	bootable, err := strconv.ParseBool(result.ParentVolumeBootable)
	if err != nil {
		return fmt.Errorf("Error converting parent volume to boolean: %v", err)
	}
	d.Set("parent_volume_bootable", bootable)

	if result.Property != compute.SnapshotPropertyCollocated {
		d.Set("collocated", false)
	} else {
		d.Set("collocated", true)
	}

	if err := setStringList(d, "tags", result.Tags); err != nil {
		return err
	}

	return nil
}

func resourceOPCStorageVolumeSnapshotDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*compute.Client).StorageVolumeSnapshots()

	name := d.Id()

	input := &compute.DeleteStorageVolumeSnapshotInput{
		Name: name,
	}

	if err := client.DeleteStorageVolumeSnapshot(input); err != nil {
		return fmt.Errorf("Error deleting storage volume snapshot '%s': %v", name, err)
	}

	return nil
}
