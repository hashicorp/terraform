package ddcloud

import (
	"github.com/DimensionDataResearch/go-dd-cloud-compute/compute"
	"github.com/hashicorp/terraform/helper/schema"
)

// resourcePropertyHelper provides commonly-used functionality for working with Terraform's schema.ResourceData.
type resourcePropertyHelper struct {
	data *schema.ResourceData
}

func propertyHelper(data *schema.ResourceData) resourcePropertyHelper {
	return resourcePropertyHelper{data}
}

func (helper resourcePropertyHelper) GetOptionalString(key string, allowEmpty bool) *string {
	value := helper.data.Get(key)
	switch typedValue := value.(type) {
	case string:
		if len(typedValue) > 0 || allowEmpty {
			return &typedValue
		}
	}

	return nil
}

func (helper resourcePropertyHelper) GetOptionalInt(key string, allowZero bool) *int {
	value := helper.data.Get(key)
	switch typedValue := value.(type) {
	case int:
		if typedValue != 0 || allowZero {
			return &typedValue
		}
	}

	return nil
}

func (helper resourcePropertyHelper) GetOptionalBool(key string) *bool {
	value := helper.data.Get(key)
	switch typedValue := value.(type) {
	case bool:
		return &typedValue
	default:
		return nil
	}
}

func (helper resourcePropertyHelper) GetServerAdditionalDisks() (disks []compute.VirtualMachineDisk) {
	value, ok := helper.data.GetOk(resourceKeyServerAdditionalDisk)
	if !ok {
		return
	}
	additionalDisks := value.(*schema.Set).List()

	disks = make([]compute.VirtualMachineDisk, len(additionalDisks))
	for index, item := range additionalDisks {
		diskProperties := item.(map[string]interface{})
		disk := &compute.VirtualMachineDisk{}

		value, ok = diskProperties[resourceKeyServerDiskID]
		if ok {
			disk.ID = stringToPtr(value.(string))
		}

		value, ok = diskProperties[resourceKeyServerDiskUnitID]
		if ok {
			disk.SCSIUnitID = value.(int)

		}
		value, ok = diskProperties[resourceKeyServerDiskSizeGB]
		if ok {
			disk.SizeGB = value.(int)
		}

		value, ok = diskProperties[resourceKeyServerDiskSpeed]
		if ok {
			disk.Speed = value.(string)
		}

		disks[index] = *disk
	}

	return
}

func (helper resourcePropertyHelper) SetServerAdditionalDisks(disks []compute.VirtualMachineDisk) {
	diskProperties := &schema.Set{F: hashDiskUnitID}

	for _, disk := range disks {
		diskProperties.Add(map[string]interface{}{
			resourceKeyServerDiskID:     *disk.ID,
			resourceKeyServerDiskSizeGB: disk.SizeGB,
			resourceKeyServerDiskUnitID: disk.SCSIUnitID,
			resourceKeyServerDiskSpeed:  disk.Speed,
		})
	}
	helper.data.Set(resourceKeyServerAdditionalDisk, diskProperties)
}
