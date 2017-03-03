package azurerm

import (
	"fmt"
	"github.com/Azure/azure-sdk-for-go/arm/disk"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	"log"
	"net/http"
	"strings"
)

func resourceArmManagedDisk() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmManagedDiskCreate,
		Read:   resourceArmManagedDiskRead,
		Update: resourceArmManagedDiskCreate,
		Delete: resourceArmManagedDiskDelete,
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

			"storage_account_type": {
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: validation.StringInSlice([]string{
					string(disk.PremiumLRS),
					string(disk.StandardLRS),
				}, true),
			},

			"create_option": {
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: validation.StringInSlice([]string{
					string(disk.Import),
					string(disk.Empty),
				}, true),
			},

			"vhd_uri": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"os_type": {
				Type:     schema.TypeString,
				Optional: true,
				ValidateFunc: validation.StringInSlice([]string{
					string(disk.Windows),
					string(disk.Linux),
				}, true),
			},

			"disk_size_gb": {
				Type:         schema.TypeInt,
				Required:     true,
				ValidateFunc: validateDiskSizeGB,
			},

			"tags": tagsSchema(),
		},
	}
}

func validateDiskSizeGB(v interface{}, k string) (ws []string, errors []error) {
	value := v.(int)
	if value < 1 || value > 1023 {
		errors = append(errors, fmt.Errorf(
			"The `disk_size_gb` can only be between 1 and 1023"))
	}
	return
}

func resourceArmManagedDiskCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	diskClient := client.diskClient

	log.Printf("[INFO] preparing arguments for Azure ARM Managed Disk creation.")

	name := d.Get("name").(string)
	location := d.Get("location").(string)
	resGroup := d.Get("resource_group_name").(string)
	tags := d.Get("tags").(map[string]interface{})
	expandedTags := expandTags(tags)

	createDisk := disk.Model{
		Name:     &name,
		Location: &location,
		Tags:     expandedTags,
	}

	storageAccountType := d.Get("storage_account_type").(string)
	osType := d.Get("os_type").(string)

	createDisk.Properties = &disk.Properties{
		AccountType: disk.StorageAccountTypes(storageAccountType),
		OsType:      disk.OperatingSystemTypes(osType),
	}

	if v := d.Get("disk_size_gb"); v != nil {
		diskSize := int32(v.(int))
		createDisk.Properties.DiskSizeGB = &diskSize
	}
	createOption := d.Get("create_option").(string)

	creationData := &disk.CreationData{
		CreateOption: disk.CreateOption(createOption),
	}

	if strings.EqualFold(createOption, string(disk.Import)) {
		if vhdUri := d.Get("vhd_uri").(string); vhdUri != "" {
			creationData.SourceURI = &vhdUri
		} else {
			return fmt.Errorf("[ERROR] vhd_uri must be specified when create_option is `%s`", disk.Import)
		}
	}

	createDisk.CreationData = creationData

	_, diskErr := diskClient.CreateOrUpdate(resGroup, name, createDisk, make(chan struct{}))
	if diskErr != nil {
		return diskErr
	}

	read, err := diskClient.Get(resGroup, name)
	if err != nil {
		return err
	}
	if read.ID == nil {
		return fmt.Errorf("[ERROR] Cannot read Managed Disk %s (resource group %s) ID", name, resGroup)
	}

	d.SetId(*read.ID)

	return resourceArmManagedDiskRead(d, meta)
}

func resourceArmManagedDiskRead(d *schema.ResourceData, meta interface{}) error {
	diskClient := meta.(*ArmClient).diskClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	name := id.Path["disks"]

	resp, err := diskClient.Get(resGroup, name)
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("[ERROR] Error making Read request on Azure Managed Disk %s (resource group %s): %s", name, resGroup, err)
	}

	d.Set("name", resp.Name)
	d.Set("resource_group_name", resGroup)
	d.Set("location", resp.Location)

	if resp.Properties != nil {
		if m, err := flattenAzureRmManagedDiskProperties(resp.Properties); err != nil {
			return fmt.Errorf("[DEBUG] Error setting disk properties: %#v", err)
		} else {
			d.Set("storage_account_type", m["storage_account_type"])
			d.Set("disk_size_gb", m["disk_size_gb"])
			if m["os_type"] != nil {
				d.Set("os_type", m["os_type"])
			}

		}
	}

	if resp.CreationData != nil {
		if m, err := flattenAzureRmManagedDiskCreationData(resp.CreationData); err != nil {
			return fmt.Errorf("[DEBUG] Error setting managed disk creation data: %#v", err)
		} else {
			d.Set("create_option", m["create_option"])
			if m["vhd_uri"] != nil {
				d.Set("vhd_uri", m["vhd_uri"])
			}
		}
	}

	flattenAndSetTags(d, resp.Tags)

	return nil
}

func resourceArmManagedDiskDelete(d *schema.ResourceData, meta interface{}) error {
	diskClient := meta.(*ArmClient).diskClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	name := id.Path["disks"]

	if _, err = diskClient.Delete(resGroup, name, make(chan struct{})); err != nil {
		return err
	}

	return nil
}

func flattenAzureRmManagedDiskProperties(properties *disk.Properties) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	result["storage_account_type"] = string(properties.AccountType)
	result["disk_size_gb"] = *properties.DiskSizeGB
	if properties.OsType != "" {
		result["os_type"] = string(properties.OsType)
	}

	return result, nil
}

func flattenAzureRmManagedDiskCreationData(creationData *disk.CreationData) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	result["create_option"] = string(creationData.CreateOption)
	if creationData.SourceURI != nil {
		result["vhd_uri"] = *creationData.SourceURI
	}

	return result, nil
}
