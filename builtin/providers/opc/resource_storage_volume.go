package opc

import (
	"fmt"
	"strconv"

	"github.com/hashicorp/go-oracle-terraform/compute"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceOPCStorageVolume() *schema.Resource {
	return &schema.Resource{
		Create: resourceOPCStorageVolumeCreate,
		Read:   resourceOPCStorageVolumeRead,
		Update: resourceOPCStorageVolumeUpdate,
		Delete: resourceOPCStorageVolumeDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"size": {
				Type:         schema.TypeInt,
				Required:     true,
				ValidateFunc: validation.IntBetween(1, 2048),
			},
			"storage_type": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  compute.StorageVolumeKindDefault,
				ValidateFunc: validation.StringInSlice([]string{
					string(compute.StorageVolumeKindDefault),
					string(compute.StorageVolumeKindLatency),
				}, true),
			},

			"snapshot": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"snapshot_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"snapshot_account": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"bootable": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
				ForceNew: true,
			},

			"image_list": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"image_list_entry": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
				Default:  -1,
			},

			"tags": tagsOptionalSchema(),

			// Computed fields
			"hypervisor": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"machine_image": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"managed": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},
			"platform": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"readonly": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},
			"status": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"storage_pool": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"uri": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func resourceOPCStorageVolumeCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*compute.Client).StorageVolumes()

	name := d.Get("name").(string)
	description := d.Get("description").(string)
	size := d.Get("size").(int)
	storageType := d.Get("storage_type").(string)
	bootable := d.Get("bootable").(bool)
	imageList := d.Get("image_list").(string)
	imageListEntry := d.Get("image_list_entry").(int)

	if bootable == true {
		if imageList == "" {
			return fmt.Errorf("Error: A Bootable Volume must have an Image List!")
		}

		if imageListEntry == -1 {
			return fmt.Errorf("Error: A Bootable Volume must have an Image List Entry!")
		}
	}

	input := compute.CreateStorageVolumeInput{
		Name:           name,
		Description:    description,
		Size:           strconv.Itoa(size),
		Properties:     []string{storageType},
		Bootable:       bootable,
		ImageList:      imageList,
		ImageListEntry: imageListEntry,
		Tags:           getStringList(d, "tags"),
	}

	if v, ok := d.GetOk("snapshot"); ok {
		input.Snapshot = v.(string)
	}
	if v, ok := d.GetOk("snapshot_account"); ok {
		input.SnapshotAccount = v.(string)
	}
	if v, ok := d.GetOk("snapshot_id"); ok {
		input.SnapshotID = v.(string)
	}

	info, err := client.CreateStorageVolume(&input)
	if err != nil {
		return fmt.Errorf("Error creating storage volume %s: %s", name, err)
	}

	d.SetId(info.Name)
	return resourceOPCStorageVolumeRead(d, meta)
}

func resourceOPCStorageVolumeUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*compute.Client).StorageVolumes()

	name := d.Id()
	description := d.Get("description").(string)
	size := d.Get("size").(int)
	storageType := d.Get("storage_type").(string)
	imageList := d.Get("image_list").(string)
	imageListEntry := d.Get("image_list_entry").(int)

	input := compute.UpdateStorageVolumeInput{
		Name:           name,
		Description:    description,
		Size:           strconv.Itoa(size),
		Properties:     []string{storageType},
		ImageList:      imageList,
		ImageListEntry: imageListEntry,
		Tags:           getStringList(d, "tags"),
	}
	_, err := client.UpdateStorageVolume(&input)
	if err != nil {
		return fmt.Errorf("Error updating storage volume %s: %s", name, err)
	}

	return resourceOPCStorageVolumeRead(d, meta)
}

func resourceOPCStorageVolumeRead(d *schema.ResourceData, meta interface{}) error {
	sv := meta.(*compute.Client).StorageVolumes()

	name := d.Id()
	input := compute.GetStorageVolumeInput{
		Name: name,
	}
	result, err := sv.GetStorageVolume(&input)
	if err != nil {
		// Volume doesn't exist
		if compute.WasNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading storage volume %s: %s", name, err)
	}

	if result == nil {
		// Volume doesn't exist
		d.SetId("")
		return nil
	}

	d.Set("name", result.Name)
	d.Set("description", result.Description)
	d.Set("storage_type", result.Properties[0])
	size, err := strconv.Atoi(result.Size)
	if err != nil {
		return err
	}
	d.Set("size", size)
	d.Set("bootable", result.Bootable)
	d.Set("image_list", result.ImageList)
	d.Set("image_list_entry", result.ImageListEntry)

	d.Set("snapshot", result.Snapshot)
	d.Set("snapshot_id", result.SnapshotID)
	d.Set("snapshot_account", result.SnapshotAccount)

	if err := setStringList(d, "tags", result.Tags); err != nil {
		return err
	}

	flattenOPCStorageVolumeComputedFields(d, result)

	return nil
}

func resourceOPCStorageVolumeDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*compute.Client).StorageVolumes()
	name := d.Id()

	input := compute.DeleteStorageVolumeInput{
		Name: name,
	}
	err := client.DeleteStorageVolume(&input)
	if err != nil {
		return fmt.Errorf("Error deleting storage volume %s: %s", name, err)
	}

	return nil
}

func flattenOPCStorageVolumeComputedFields(d *schema.ResourceData, result *compute.StorageVolumeInfo) {
	d.Set("hypervisor", result.Hypervisor)
	d.Set("machine_image", result.MachineImage)
	d.Set("managed", result.Managed)
	d.Set("platform", result.Platform)
	d.Set("readonly", result.ReadOnly)
	d.Set("status", result.Status)
	d.Set("storage_pool", result.StoragePool)
	d.Set("uri", result.URI)
}
