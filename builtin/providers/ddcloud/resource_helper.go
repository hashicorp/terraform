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

func (helper resourcePropertyHelper) SetPartial(key string) {
	helper.data.SetPartial(key)
}

func (helper resourcePropertyHelper) GetTags(key string) (tags []compute.Tag) {
	value, ok := helper.data.GetOk(key)
	if !ok {
		return
	}
	tagData := value.(*schema.Set).List()

	tags = make([]compute.Tag, len(tagData))
	for index, item := range tagData {
		tagProperties := item.(map[string]interface{})
		tag := &compute.Tag{}

		value, ok = tagProperties[resourceKeyServerTagName] // TODO: Move this out of servers.
		if ok {
			tag.Name = value.(string)
		}

		value, ok = tagProperties[resourceKeyServerTagValue] // TODO: Move this out of servers.
		if ok {
			tag.Value = value.(string)
		}

		tags[index] = *tag
	}

	return
}

func (helper resourcePropertyHelper) SetTags(key string, tags []compute.Tag) {
	tagProperties := &schema.Set{F: hashServerTag}

	for _, tag := range tags {
		tagProperties.Add(map[string]interface{}{
			resourceKeyServerTagName:  tag.Name,
			resourceKeyServerTagValue: tag.Value,
		})
	}
	helper.data.Set(key, tagProperties)
}

func (helper resourcePropertyHelper) GetServerDisks(key string) (disks []compute.VirtualMachineDisk) {
	value, ok := helper.data.GetOk(key)
	if !ok {
		return
	}
	serverDisks := value.(*schema.Set).List()

	disks = make([]compute.VirtualMachineDisk, len(serverDisks))
	for index, item := range serverDisks {
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

func (helper resourcePropertyHelper) SetServerDisks(key string, disks []compute.VirtualMachineDisk) {
	diskProperties := &schema.Set{F: hashDiskUnitID}

	for _, disk := range disks {
		diskProperties.Add(map[string]interface{}{
			resourceKeyServerDiskID:     *disk.ID,
			resourceKeyServerDiskSizeGB: disk.SizeGB,
			resourceKeyServerDiskUnitID: disk.SCSIUnitID,
			resourceKeyServerDiskSpeed:  disk.Speed,
		})
	}
	helper.data.Set(key, diskProperties)
}
