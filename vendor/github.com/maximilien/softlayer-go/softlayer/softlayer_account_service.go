package softlayer

import (
	datatypes "github.com/maximilien/softlayer-go/data_types"
)

type SoftLayer_Account_Service interface {
	Service

	GetAccountStatus() (datatypes.SoftLayer_Account_Status, error)
	GetVirtualGuests() ([]datatypes.SoftLayer_Virtual_Guest, error)
	GetVirtualGuestsByFilter(filters string) ([]datatypes.SoftLayer_Virtual_Guest, error)
	GetNetworkStorage() ([]datatypes.SoftLayer_Network_Storage, error)
	GetIscsiNetworkStorage() ([]datatypes.SoftLayer_Network_Storage, error)
	GetIscsiNetworkStorageWithFilter(filter string) ([]datatypes.SoftLayer_Network_Storage, error)
	GetVirtualDiskImages() ([]datatypes.SoftLayer_Virtual_Disk_Image, error)
	GetVirtualDiskImagesWithFilter(filters string) ([]datatypes.SoftLayer_Virtual_Disk_Image, error)
	GetSshKeys() ([]datatypes.SoftLayer_Security_Ssh_Key, error)
	GetBlockDeviceTemplateGroups() ([]datatypes.SoftLayer_Virtual_Guest_Block_Device_Template_Group, error)
	GetBlockDeviceTemplateGroupsWithFilter(filters string) ([]datatypes.SoftLayer_Virtual_Guest_Block_Device_Template_Group, error)
	GetDatacentersWithSubnetAllocations() ([]datatypes.SoftLayer_Location, error)
	GetHardware() ([]datatypes.SoftLayer_Hardware, error)
	GetDomains() ([]datatypes.SoftLayer_Dns_Domain, error)
}
