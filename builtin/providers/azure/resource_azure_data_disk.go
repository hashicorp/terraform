package azure

import (
	"fmt"
	"log"
	"time"

	"github.com/Azure/azure-sdk-for-go/management"
	"github.com/Azure/azure-sdk-for-go/management/virtualmachinedisk"
	"github.com/hashicorp/terraform/helper/schema"
)

const dataDiskBlobStorageURL = "http://%s.blob.core.windows.net/disks/%s.vhd"

func resourceAzureDataDisk() *schema.Resource {
	return &schema.Resource{
		Create: resourceAzureDataDiskCreate,
		Read:   resourceAzureDataDiskRead,
		Update: resourceAzureDataDiskUpdate,
		Delete: resourceAzureDataDiskDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"label": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"lun": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			},

			"size": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},

			"caching": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "None",
			},

			"storage_service_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"media_link": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"source_media_link": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"virtual_machine": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceAzureDataDiskCreate(d *schema.ResourceData, meta interface{}) error {
	mc := meta.(*Client).mgmtClient
	vmDiskClient := meta.(*Client).vmDiskClient

	if err := verifyDataDiskParameters(d); err != nil {
		return err
	}

	lun := d.Get("lun").(int)
	vm := d.Get("virtual_machine").(string)

	label := d.Get("label").(string)
	if label == "" {
		label = fmt.Sprintf("%s-%d", vm, lun)
	}

	p := virtualmachinedisk.CreateDataDiskParameters{
		DiskLabel:           label,
		Lun:                 lun,
		LogicalDiskSizeInGB: d.Get("size").(int),
		HostCaching:         hostCaching(d),
		MediaLink:           mediaLink(d),
		SourceMediaLink:     d.Get("source_media_link").(string),
	}

	if name, ok := d.GetOk("name"); ok {
		p.DiskName = name.(string)
	}

	log.Printf("[DEBUG] Adding data disk %d to instance: %s", lun, vm)
	req, err := vmDiskClient.AddDataDisk(vm, vm, vm, p)
	if err != nil {
		return fmt.Errorf("Error adding data disk %d to instance %s: %s", lun, vm, err)
	}

	// Wait until the data disk is added
	if err := mc.WaitForOperation(req, nil); err != nil {
		return fmt.Errorf(
			"Error waiting for data disk %d to be added to instance %s: %s", lun, vm, err)
	}

	log.Printf("[DEBUG] Retrieving data disk %d from instance %s", lun, vm)
	disk, err := vmDiskClient.GetDataDisk(vm, vm, vm, lun)
	if err != nil {
		return fmt.Errorf("Error retrieving data disk %d from instance %s: %s", lun, vm, err)
	}

	d.SetId(disk.DiskName)

	return resourceAzureDataDiskRead(d, meta)
}

func resourceAzureDataDiskRead(d *schema.ResourceData, meta interface{}) error {
	vmDiskClient := meta.(*Client).vmDiskClient

	lun := d.Get("lun").(int)
	vm := d.Get("virtual_machine").(string)

	log.Printf("[DEBUG] Retrieving data disk: %s", d.Id())
	datadisk, err := vmDiskClient.GetDataDisk(vm, vm, vm, lun)
	if err != nil {
		if management.IsResourceNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error retrieving data disk %s: %s", d.Id(), err)
	}

	d.Set("name", datadisk.DiskName)
	d.Set("label", datadisk.DiskLabel)
	d.Set("lun", datadisk.Lun)
	d.Set("size", datadisk.LogicalDiskSizeInGB)
	d.Set("caching", datadisk.HostCaching)
	d.Set("media_link", datadisk.MediaLink)

	log.Printf("[DEBUG] Retrieving disk: %s", d.Id())
	disk, err := vmDiskClient.GetDisk(d.Id())
	if err != nil {
		return fmt.Errorf("Error retrieving disk %s: %s", d.Id(), err)
	}

	d.Set("virtual_machine", disk.AttachedTo.RoleName)

	return nil
}

func resourceAzureDataDiskUpdate(d *schema.ResourceData, meta interface{}) error {
	mc := meta.(*Client).mgmtClient
	vmDiskClient := meta.(*Client).vmDiskClient

	lun := d.Get("lun").(int)
	vm := d.Get("virtual_machine").(string)

	if d.HasChange("lun") || d.HasChange("size") || d.HasChange("virtual_machine") {
		olun, _ := d.GetChange("lun")
		ovm, _ := d.GetChange("virtual_machine")

		log.Printf("[DEBUG] Detaching data disk: %s", d.Id())
		req, err := vmDiskClient.
			DeleteDataDisk(ovm.(string), ovm.(string), ovm.(string), olun.(int), false)
		if err != nil {
			return fmt.Errorf("Error detaching data disk %s: %s", d.Id(), err)
		}

		// Wait until the data disk is detached
		if err := mc.WaitForOperation(req, nil); err != nil {
			return fmt.Errorf(
				"Error waiting for data disk %s to be detached: %s", d.Id(), err)
		}

		log.Printf("[DEBUG] Verifying data disk %s is properly detached...", d.Id())
		for i := 0; i < 6; i++ {
			disk, err := vmDiskClient.GetDisk(d.Id())
			if err != nil {
				return fmt.Errorf("Error retrieving disk %s: %s", d.Id(), err)
			}

			// Check if the disk is really detached
			if disk.AttachedTo.RoleName == "" {
				break
			}

			// If not, wait 30 seconds and try it again...
			time.Sleep(time.Duration(30 * time.Second))
		}

		if d.HasChange("size") {
			p := virtualmachinedisk.UpdateDiskParameters{
				Name:            d.Id(),
				Label:           d.Get("label").(string),
				ResizedSizeInGB: d.Get("size").(int),
			}

			log.Printf("[DEBUG] Updating disk: %s", d.Id())
			req, err := vmDiskClient.UpdateDisk(d.Id(), p)
			if err != nil {
				return fmt.Errorf("Error updating disk %s: %s", d.Id(), err)
			}

			// Wait until the disk is updated
			if err := mc.WaitForOperation(req, nil); err != nil {
				return fmt.Errorf(
					"Error waiting for disk %s to be updated: %s", d.Id(), err)
			}
		}

		p := virtualmachinedisk.CreateDataDiskParameters{
			DiskName:    d.Id(),
			Lun:         lun,
			HostCaching: hostCaching(d),
			MediaLink:   mediaLink(d),
		}

		log.Printf("[DEBUG] Attaching data disk: %s", d.Id())
		req, err = vmDiskClient.AddDataDisk(vm, vm, vm, p)
		if err != nil {
			return fmt.Errorf("Error attaching data disk %s to instance %s: %s", d.Id(), vm, err)
		}

		// Wait until the data disk is attached
		if err := mc.WaitForOperation(req, nil); err != nil {
			return fmt.Errorf(
				"Error waiting for data disk %s to be attached to instance %s: %s", d.Id(), vm, err)
		}

		// Make sure we return here since all possible changes are
		// already updated if we reach this point
		return nil
	}

	if d.HasChange("caching") {
		p := virtualmachinedisk.UpdateDataDiskParameters{
			DiskName:    d.Id(),
			Lun:         lun,
			HostCaching: hostCaching(d),
			MediaLink:   mediaLink(d),
		}

		log.Printf("[DEBUG] Updating data disk: %s", d.Id())
		req, err := vmDiskClient.UpdateDataDisk(vm, vm, vm, lun, p)
		if err != nil {
			return fmt.Errorf("Error updating data disk %s: %s", d.Id(), err)
		}

		// Wait until the data disk is updated
		if err := mc.WaitForOperation(req, nil); err != nil {
			return fmt.Errorf(
				"Error waiting for data disk %s to be updated: %s", d.Id(), err)
		}
	}

	return resourceAzureDataDiskRead(d, meta)
}

func resourceAzureDataDiskDelete(d *schema.ResourceData, meta interface{}) error {
	mc := meta.(*Client).mgmtClient
	vmDiskClient := meta.(*Client).vmDiskClient

	lun := d.Get("lun").(int)
	vm := d.Get("virtual_machine").(string)

	// If a name was not supplied, it means we created a new emtpy disk and we now want to
	// delete that disk again. Otherwise we only want to detach the disk and keep the blob.
	_, removeBlob := d.GetOk("name")

	log.Printf("[DEBUG] Detaching data disk %s with removeBlob = %t", d.Id(), removeBlob)
	req, err := vmDiskClient.DeleteDataDisk(vm, vm, vm, lun, removeBlob)
	if err != nil {
		return fmt.Errorf(
			"Error detaching data disk %s with removeBlob = %t: %s", d.Id(), removeBlob, err)
	}

	// Wait until the data disk is detached and optionally deleted
	if err := mc.WaitForOperation(req, nil); err != nil {
		return fmt.Errorf(
			"Error waiting for data disk %s to be detached with removeBlob = %t: %s",
			d.Id(), removeBlob, err)
	}

	d.SetId("")

	return nil
}

func hostCaching(d *schema.ResourceData) virtualmachinedisk.HostCachingType {
	switch d.Get("caching").(string) {
	case "ReadOnly":
		return virtualmachinedisk.HostCachingTypeReadOnly
	case "ReadWrite":
		return virtualmachinedisk.HostCachingTypeReadWrite
	default:
		return virtualmachinedisk.HostCachingTypeNone
	}
}

func mediaLink(d *schema.ResourceData) string {
	mediaLink, ok := d.GetOk("media_link")
	if ok {
		return mediaLink.(string)
	}

	name, ok := d.GetOk("name")
	if !ok {
		name = fmt.Sprintf("%s-%d", d.Get("virtual_machine").(string), d.Get("lun").(int))
	}

	return fmt.Sprintf(dataDiskBlobStorageURL, d.Get("storage_service_name").(string), name.(string))
}

func verifyDataDiskParameters(d *schema.ResourceData) error {
	caching := d.Get("caching").(string)
	if caching != "None" && caching != "ReadOnly" && caching != "ReadWrite" {
		return fmt.Errorf(
			"Invalid caching type %s! Valid options are 'None', 'ReadOnly' and 'ReadWrite'.", caching)
	}

	if _, ok := d.GetOk("media_link"); !ok {
		if _, ok := d.GetOk("storage_service_name"); !ok {
			return fmt.Errorf("If not supplying 'media_link', you must supply 'storage'.")
		}
	}

	return nil
}
