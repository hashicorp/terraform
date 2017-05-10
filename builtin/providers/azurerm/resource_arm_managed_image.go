package azurerm

import (
	"fmt"
	"log"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/arm/compute"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceArmManagedImage() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmManagedImageCreate,
		Read:   resourceArmManagedImageRead,
		Update: resourceArmManagedImageCreate,
		Delete: resourceArmManagedImageDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"location": locationSchema(),

			"resource_group_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"source_virtual_machine_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"os_disk.os_type"},
			},

			"os_disk": {
				Type:     schema.TypeSet,
				Optional: true,
				MaxItems: 1,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{

						"os_type": {
							Type:     schema.TypeString,
							Optional: true,
							ValidateFunc: validation.StringInSlice([]string{
								string(compute.Linux),
								string(compute.Windows),
							}, true),
						},

						"os_state": {
							Type:     schema.TypeString,
							Optional: true,
							ValidateFunc: validation.StringInSlice([]string{
								string(compute.Generalized),
								string(compute.Specialized),
							}, true),
						},

						"managed_disk_id": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"blob_uri": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
							ForceNew: true,
						},

						"caching": {
							Type:     schema.TypeString,
							Optional: true,
							ValidateFunc: validation.StringInSlice([]string{
								string(compute.None),
								string(compute.ReadOnly),
								string(compute.ReadWrite),
							}, true),
						},

						"size_gb": {
							Type:         schema.TypeInt,
							Optional:     true,
							ValidateFunc: validateDiskSizeGB,
						},
					},
				},
			},

			"data_disk": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{

						"lun": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
						},

						"managed_disk_id": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"blob_uri": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
							ForceNew: true,
						},

						"caching": {
							Type:     schema.TypeString,
							Optional: true,
							ValidateFunc: validation.StringInSlice([]string{
								string(compute.None),
								string(compute.ReadOnly),
								string(compute.ReadWrite),
							}, true),
						},

						"size_gb": {
							Type:         schema.TypeInt,
							Optional:     true,
							ValidateFunc: validateDiskSizeGB,
						},
					},
				},
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceArmManagedImageCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	imageClient := client.imageClient

	log.Printf("[INFO] preparing arguments for Azure ARM Image creation.")

	name := d.Get("name").(string)
	location := d.Get("location").(string)
	resGroup := d.Get("resource_group_name").(string)
	tags := d.Get("tags").(map[string]interface{})
	expandedTags := expandTags(tags)

	osDisk, err := expandAzureRmImageOsDisk(d)
	if err != nil {
		return err
	}

	dataDisks, err := expandAzureRmImageDataDisks(d)
	if err != nil {
		return err
	}

	storageProfile := compute.ImageStorageProfile{
		OsDisk:    osDisk,
		DataDisks: &dataDisks,
	}

	sourceVM := compute.SubResource{}
	if v, ok := d.GetOk("source_virtual_machine_id"); ok {
		vmID := v.(string)
		sourceVM = compute.SubResource{
			ID: &vmID,
		}
	}

	properties := compute.ImageProperties{}
	//either source VM or storage profile can be specified, but not both
	if (compute.SubResource{}) == sourceVM {
		properties = compute.ImageProperties{
			StorageProfile: &storageProfile,
		}
	} else {
		properties = compute.ImageProperties{
			SourceVirtualMachine: &sourceVM,
		}
	}

	createImage := compute.Image{
		Name:            &name,
		Location:        &location,
		Tags:            expandedTags,
		ImageProperties: &properties,
	}

	_, imageErr := imageClient.CreateOrUpdate(resGroup, name, createImage, make(chan struct{}))
	if imageErr != nil {
		return imageErr
	}

	read, err := imageClient.Get(resGroup, name, "")
	if err != nil {
		return err
	}
	if read.ID == nil {
		return fmt.Errorf("[ERROR] Cannot read Image %s (resource group %s) ID", name, resGroup)
	}

	d.SetId(*read.ID)

	return resourceArmManagedImageRead(d, meta)
}

func resourceArmManagedImageRead(d *schema.ResourceData, meta interface{}) error {
	imageClient := meta.(*ArmClient).imageClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	name := id.Path["images"]

	resp, err := imageClient.Get(resGroup, name, "")
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("[ERROR] Error making Read request on Azure Image %s (resource group %s): %s", name, resGroup, err)
	}

	d.Set("name", resp.Name)
	d.Set("resource_group_name", resGroup)
	d.Set("location", resp.Location)

	if resp.SourceVirtualMachine != nil {
		flattenAzureRmSourceVMProperties(d, resp.SourceVirtualMachine)
	}

	if resp.StorageProfile != nil {
		if err := d.Set("os_disk", flattenAzureRmStorageProfileOsDisk(d, resp.StorageProfile)); err != nil {
			return fmt.Errorf("[DEBUG] Error setting Managed Images OS Disk error: %#v", err)
		}

		if resp.StorageProfile.DataDisks != nil {
			if err := d.Set("data_disk", flattenAzureRmStorageProfileDataDisks(d, resp.StorageProfile)); err != nil {
				return fmt.Errorf("[DEBUG] Error setting Managed Images Data Disks error: %#v", err)
			}
		}
	}

	flattenAndSetTags(d, resp.Tags)

	return nil
}

func resourceArmManagedImageDelete(d *schema.ResourceData, meta interface{}) error {
	imageClient := meta.(*ArmClient).imageClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	name := id.Path["images"]

	if _, err = imageClient.Delete(resGroup, name, make(chan struct{})); err != nil {
		return err
	}

	return nil
}

func flattenAzureRmSourceVMProperties(d *schema.ResourceData, properties *compute.SubResource) {
	if properties.ID != nil {
		d.Set("source_virtual_machine_id", *properties.ID)
	}
}

func flattenAzureRmStorageProfileOsDisk(d *schema.ResourceData, storageProfile *compute.ImageStorageProfile) []interface{} {
	result := make(map[string]interface{})
	if storageProfile.OsDisk != nil {
		osDisk := *storageProfile.OsDisk
		result["os_type"] = osDisk.OsType
		result["os_state"] = osDisk.OsState
		if osDisk.ManagedDisk != nil {
			result["managed_disk_id"] = *osDisk.ManagedDisk.ID
		}
		result["blob_uri"] = *osDisk.BlobURI
		result["caching"] = osDisk.Caching
	}

	return []interface{}{result}
}

func flattenAzureRmStorageProfileDataDisks(d *schema.ResourceData, storageProfile *compute.ImageStorageProfile) []interface{} {
	disks := storageProfile.DataDisks
	result := make([]interface{}, len(*disks))
	for i, disk := range *disks {
		l := make(map[string]interface{})
		if disk.ManagedDisk != nil {
			l["managed_disk_id"] = *disk.ManagedDisk.ID
		}
		l["blob_uri"] = disk.BlobURI
		l["caching"] = string(disk.Caching)
		if disk.DiskSizeGB != nil {
			l["size_gb"] = *disk.DiskSizeGB
		}
		l["lun"] = *disk.Lun

		result[i] = l
	}
	return result
}

func expandAzureRmImageOsDisk(d *schema.ResourceData) (*compute.ImageOSDisk, error) {

	osDisk := &compute.ImageOSDisk{}
	disks := d.Get("os_disk").(*schema.Set).List()

	if len(disks) > 0 {
		config := disks[0].(map[string]interface{})

		if v := config["os_type"].(string); v != "" {
			osType := compute.OperatingSystemTypes(v)
			osDisk.OsType = osType
		}

		if v := config["os_state"].(string); v != "" {
			osState := compute.OperatingSystemStateTypes(v)
			osDisk.OsState = osState
		}

		managedDisk := &compute.SubResource{}
		managedDiskID := config["managed_disk_id"].(string)
		if managedDiskID != "" {
			managedDisk.ID = &managedDiskID
			osDisk.ManagedDisk = managedDisk
		}

		blobURI := config["blob_uri"].(string)
		osDisk.BlobURI = &blobURI
		if v := config["caching"].(string); v != "" {
			caching := compute.CachingTypes(v)
			osDisk.Caching = caching
		}

		diskSize := int32(0)
		if size := config["size_gb"]; size != 0 {
			diskSize = int32(size.(int))
			osDisk.DiskSizeGB = &diskSize
		}
	}

	return osDisk, nil
}

func expandAzureRmImageDataDisks(d *schema.ResourceData) ([]compute.ImageDataDisk, error) {

	disks := d.Get("data_disk").([]interface{})

	dataDisks := make([]compute.ImageDataDisk, 0, len(disks))
	for _, diskConfig := range disks {
		config := diskConfig.(map[string]interface{})

		managedDiskID := d.Get("managed_disk_id").(string)
		blobURI := d.Get("blob_uri").(string)

		lun := int32(config["lun"].(int))

		diskSize := int32(0)
		if size := d.Get("size_gb"); size != 0 {
			diskSize = int32(size.(int))
		}

		dataDisk := compute.ImageDataDisk{
			Lun:        &lun,
			BlobURI:    &blobURI,
			DiskSizeGB: &diskSize,
		}

		if v := d.Get("caching").(string); v != "" {
			caching := compute.CachingTypes(v)
			dataDisk.Caching = caching
		}

		managedDisk := &compute.SubResource{}
		if managedDiskID != "" {
			managedDisk.ID = &managedDiskID
			dataDisk.ManagedDisk = managedDisk
		}

		dataDisks = append(dataDisks, dataDisk)
	}

	return dataDisks, nil

}
