package azurerm

import (
	"github.com/Azure/azure-sdk-for-go/arm/disk"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	"fmt"
	"log"
	"strings"
	"net/http"
)

func resourceArmDisk() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmDiskCreate,
		Read:   resourceArmDiskRead,
		Update: resourceArmDiskCreate,
		Delete: resourceArmDiskDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type: schema.TypeString,
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
				Type: schema.TypeString,
				Required: true,
				ValidateFunc: validation.StringInSlice([]string{
					string(disk.PremiumLRS),
					string(disk.StandardLRS),
				}, true),
			},

			"create_option": {
				Type: schema.TypeString,
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
				Type: schema.TypeString,
				Optional: true,
				ValidateFunc: validation.StringInSlice([]string{
					string(disk.Windows),
					string(disk.Linux),
				}, true),
			},

			"disk_size_gb": {
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
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

func resourceArmDiskCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	diskClient := client.diskClient

	log.Printf("[INFO] preparing arguments for Azure ARM Disk creation.")

	name := d.Get("name").(string)
	location := d.Get("location").(string)
	resGroup := d.Get("resource_group_name").(string)
	tags := d.Get("tags").(map[string]interface{})
	expandedTags := expandTags(tags)

	createDisk := &disk.Model {
		Name:     &name,
		Location: &location,
		Tags:     expandedTags,
	}

	storageAccountType := d.Get("storage_account_type").(string)
	osType := d.Get("os_type").(string)
	diskSize := d.Get("disk_size_db").(int)

	createDisk.Properties = &disk.Properties {
		AccountType: &storageAccountType,
		OsType:      &osType,
		DiskSizeGB:  &diskSize,
	}

	createOption := d.Get("create_option").(string)

	creationData := &disk.CreationData{
		CreateOption: disk.CreateOption(createOption),
	}

	if strings.EqualFold(createOption, disk.Import) {
		if vhdUri := d.Get("vhd_uri").(string); vhdUri != "" {
			creationData.SourceURI = vhdUri;
		} else {
			return nil, fmt.Errorf("[ERROR] vhd_uri must be specified when create_option is `%s`", disk.Import)
		}
	}

	createDisk.CreationData = creationData

	_, diskErr := diskClient.CreateOrUpdate(resGroup, name, createDisk, make(chan struct{}))
	if diskErr != nil {
		return diskErr
	}

	read, err := diskClient.Get(resGroup, name)
	if err != nil{
		return err
	}
	if read.ID == nil {
		return fmt.Errorf("[ERROR] Cannot read Disk %s (resource group %s) ID", name, resGroup)
	}

	d.SetId(*read.ID)

	return resourceArmDiskRead(d, meta)
}

func resourceArmDiskRead(d *schema.ResourceData, meta interface{}) error {
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
		return fmt.Errorf("[ERROR] Error making Read request on Azure Disk %s (resource group %s): %s", name, resGroup, err)
	}

	d.Set("name", resp.Name)
	d.Set("resource_group_name", resGroup)
	d.Set("location", resp.Location)

	if resp.Properties != nil {
		if m, err := flattenAzureRmDiskProperties(resp.Properties); err != nil {
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
		if m, err := flattenAzureRmDiskCreationData(resp.CreationData); err != nil {
			return fmt.Errorf("[DEBUG] Error setting disk creation data: %#v", err)
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

func resourceArmDiskDelete(d *schema.ResourceData, meta interface{}) error {
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

func flattenAzureRmDiskProperties(properties *disk.Properties) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	result["storage_account_type"] = *properties.AccountType
	result["disk_size_gb"] = *properties.DiskSizeGB
	if properties.OsType != nil {
		result["os_type"] = *properties.OsType
	}

	return result, nil
}

func flattenAzureRmDiskCreationData(creationData *disk.CreationData) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	result["create_option"] = *creationData.CreateOption
	if creationData.SourceURI != nil {
		result["vhd_uri"] = *creationData.SourceURI
	}

	return result, nil
}