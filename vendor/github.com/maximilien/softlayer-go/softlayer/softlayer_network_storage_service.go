package softlayer

import (
	datatypes "github.com/maximilien/softlayer-go/data_types"
)

type SoftLayer_Network_Storage_Service interface {
	Service

	DeleteObject(volumeId int) (bool, error)

	CreateNetworkStorage(size int, capacity int, location string, userHourlyPricing bool) (datatypes.SoftLayer_Network_Storage, error)
	DeleteNetworkStorage(volumeId int, immediateCancellationFlag bool) error
	GetNetworkStorage(volumeId int) (datatypes.SoftLayer_Network_Storage, error)
	GetBillingItem(volumeId int) (datatypes.SoftLayer_Billing_Item, error)
	HasAllowedVirtualGuest(volumeId int, vmId int) (bool, error)
	HasAllowedHardware(volumeId int, vmId int) (bool, error)
	AttachNetworkStorageToVirtualGuest(virtualGuest datatypes.SoftLayer_Virtual_Guest, volumeId int) (bool, error)
	DetachNetworkStorageFromVirtualGuest(virtualGuest datatypes.SoftLayer_Virtual_Guest, volumeId int) error
	AttachNetworkStorageToHardware(hardware datatypes.SoftLayer_Hardware, volumeId int) (bool, error)
	DetachNetworkStorageFromHardware(hardware datatypes.SoftLayer_Hardware, volumeId int) error
}
