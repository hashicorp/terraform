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

			"bootable": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
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
					},
				},
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

	input := compute.CreateStorageVolumeInput{
		Name:        name,
		Description: description,
		Size:        strconv.Itoa(size),
		Properties:  []string{storageType},
		Tags:        getStringList(d, "tags"),
	}

	expandOPCStorageVolumeOptionalFields(d, input)

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

	input := compute.UpdateStorageVolumeInput{
		Name:        name,
		Description: description,
		Size:        strconv.Itoa(size),
		Properties:  []string{storageType},
		Tags:        getStringList(d, "tags"),
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
	d.Set("storage", result.Properties[0])
	size, err := strconv.Atoi(result.Size)
	if err != nil {
		return err
	}
	d.Set("size", size)

	if err := setStringList(d, "tags", result.Tags); err != nil {
		return err
	}

	flattenOPCStorageVolumeOptionalFields(d, result)

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

func expandOPCStorageVolumeOptionalFields(d *schema.ResourceData, input compute.CreateStorageVolumeInput) {
	value, exists := d.GetOk("bootable")
	input.Bootable = exists
	if exists {
		configs := value.([]interface{})
		config := configs[0].(map[string]interface{})

		input.ImageList = config["image_list"].(string)
		input.ImageListEntry = config["image_list_entry"].(int)
	}
}

func flattenOPCStorageVolumeOptionalFields(d *schema.ResourceData, result *compute.StorageVolumeInfo) {
	d.Set("bootable", result.Bootable)
	d.Set("image_list", result.ImageList)
	d.Set("image_list_entry", result.ImageListEntry)
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
