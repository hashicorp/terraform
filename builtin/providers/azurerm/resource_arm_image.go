package azurerm

import (
	"fmt"
	"log"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/arm/compute"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceArmImage() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmImageCreateUpdate,
		Read:   resourceArmImageRead,
		Update: resourceArmImageCreateUpdate,
		Delete: resourceArmImageDelete,
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
				ConflictsWith: []string{"os_disk_os_type"},
			},

			"os_disk_os_type": {
				Type:     schema.TypeString,
				Optional: true,
				ValidateFunc: validation.StringInSlice([]string{
					string(compute.Linux),
					string(compute.Windows),
				}, true),
				DiffSuppressFunc: ignoreCaseDiffSuppressFunc,
			},

			"os_disk_os_state": {
				Type:     schema.TypeString,
				Optional: true,
				ValidateFunc: validation.StringInSlice([]string{
					string(compute.Generalized),
					string(compute.Specialized),
				}, true),
				DiffSuppressFunc: ignoreCaseDiffSuppressFunc,
			},

			"os_disk_managed_disk_id": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"os_disk_blob_uri": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"os_disk_caching": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "None",
				ValidateFunc: validation.StringInSlice([]string{
					string(compute.None),
					string(compute.ReadOnly),
					string(compute.ReadWrite),
				}, true),
				DiffSuppressFunc: ignoreCaseDiffSuppressFunc,
			},

			"os_disk_size_gb": {
				Type:     schema.TypeInt,
				Optional: true,
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
						},

						"caching": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "None",
							ValidateFunc: validation.StringInSlice([]string{
								string(compute.None),
								string(compute.ReadOnly),
								string(compute.ReadWrite),
							}, true),
							DiffSuppressFunc: ignoreCaseDiffSuppressFunc,
						},

						"size_gb": {
							Type:     schema.TypeInt,
							Optional: true,
						},
					},
				},
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceArmImageCreateUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	imageClient := client.imageClient

	log.Printf("[INFO] preparing arguments for Azure ARM Image creation.")

	name := d.Get("name").(string)
	location := d.Get("location").(string)
	resGroup := d.Get("resource_group_name").(string)
	tags := d.Get("tags").(map[string]interface{})
	expandedTags := expandTags(tags)

	osDisk := &compute.ImageOSDisk{}
	if v, ok := d.Get("os_disk_os_type").(string); ok {
		osType := compute.OperatingSystemTypes(v)
		osDisk.OsType = osType
	}

	if v, ok := d.Get("os_disk_os_state").(string); ok {
		osState := compute.OperatingSystemStateTypes(v)
		osDisk.OsState = osState
	}

	managedDisk := &compute.SubResource{}
	if managedDiskID, _ := d.Get("os_disk_managed_disk_id").(string); managedDiskID != "" {
		managedDisk.ID = &managedDiskID
		osDisk.ManagedDisk = managedDisk
	}

	blobURI := d.Get("os_disk_blob_uri").(string)
	if blobURI != "" {
		osDisk.BlobURI = &blobURI
	}
	if v := d.Get("os_disk_caching").(string); v != "" {
		caching := compute.CachingTypes(v)
		osDisk.Caching = caching
	}

	diskSize := int32(0)
	if size := d.Get("os_disk_size_gb"); size != 0 {
		diskSize = int32(size.(int))
		osDisk.DiskSizeGB = &diskSize
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
	err = <-imageErr
	if err != nil {
		return err
	}

	read, err := imageClient.Get(resGroup, name, "")
	if err != nil {
		return err
	}
	if read.ID == nil {
		return fmt.Errorf("[ERROR] Cannot read Image %s (resource group %s) ID", name, resGroup)
	}

	d.SetId(*read.ID)

	return resourceArmImageRead(d, meta)
}

func resourceArmImageRead(d *schema.ResourceData, meta interface{}) error {
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
		return fmt.Errorf("[ERROR] Error making Read request on AzureRM Image %s (resource group %s): %s", name, resGroup, err)
	}

	d.Set("name", resp.Name)
	d.Set("resource_group_name", resGroup)
	d.Set("location", resp.Location)

	//either source VM or storage profile can be specified, but not both
	if resp.SourceVirtualMachine != nil {
		flattenAzureRmSourceVMProperties(d, resp.SourceVirtualMachine)
	} else if resp.StorageProfile != nil {
		if resp.StorageProfile.OsDisk != nil {
			d.Set("os_disk_os_type", resp.StorageProfile.OsDisk.OsType)
			d.Set("os_disk_os_state", resp.StorageProfile.OsDisk.OsState)

			if resp.StorageProfile.OsDisk.ManagedDisk != nil {
				d.Set("os_disk_managed_disk_id", *resp.StorageProfile.OsDisk.ManagedDisk.ID)
			}
			if resp.StorageProfile.OsDisk.BlobURI != nil {
				d.Set("os_disk_blob_uri", *resp.StorageProfile.OsDisk.BlobURI)
			}
			d.Set("os_disk_caching", resp.StorageProfile.OsDisk.Caching)
			if resp.StorageProfile.OsDisk.DiskSizeGB != nil {
				d.Set("os_disk_size_gb", *resp.StorageProfile.OsDisk.DiskSizeGB)
			}
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

func resourceArmImageDelete(d *schema.ResourceData, meta interface{}) error {
	imageClient := meta.(*ArmClient).imageClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	name := id.Path["images"]

	_, deleteErr := imageClient.Delete(resGroup, name, make(chan struct{}))
	err = <-deleteErr
	if err != nil {
		return err
	}

	return nil
}

func flattenAzureRmSourceVMProperties(d *schema.ResourceData, properties *compute.SubResource) {
	if properties.ID != nil {
		d.Set("source_virtual_machine_id", properties.ID)
	}
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

func expandAzureRmImageDataDisks(d *schema.ResourceData) ([]compute.ImageDataDisk, error) {

	disks := d.Get("data_disk").([]interface{})

	dataDisks := make([]compute.ImageDataDisk, 0, len(disks))
	for _, diskConfig := range disks {
		config := diskConfig.(map[string]interface{})

		managedDiskID := d.Get("managed_disk_id").(string)
		blobURI := d.Get("blob_uri").(string)
		lun := int32(config["lun"].(int))

		dataDisk := compute.ImageDataDisk{
			Lun:     &lun,
			BlobURI: &blobURI,
		}

		if size := d.Get("size_gb"); size != 0 {
			diskSize := int32(size.(int))
			dataDisk.DiskSizeGB = &diskSize
		}

		if v := d.Get("caching").(string); v != "" {
			caching := compute.CachingTypes(v)
			dataDisk.Caching = caching
		}

		if managedDiskID != "" {
			managedDisk := &compute.SubResource{}
			managedDisk.ID = &managedDiskID
			dataDisk.ManagedDisk = managedDisk
		}

		dataDisks = append(dataDisks, dataDisk)
	}

	return dataDisks, nil

}
