package softlayer

import (
	datatypes "github.com/maximilien/softlayer-go/data_types"
)

type SoftLayer_Hardware_Service interface {
	Service

	AllowAccessToNetworkStorage(id int, storage datatypes.SoftLayer_Network_Storage) (bool, error)

	CreateObject(template datatypes.SoftLayer_Hardware_Template) (datatypes.SoftLayer_Hardware, error)

	FindByIpAddress(ipAddress string) (datatypes.SoftLayer_Hardware, error)

	GetObject(id int) (datatypes.SoftLayer_Hardware, error)
	GetAllowedHost(id int) (datatypes.SoftLayer_Network_Storage_Allowed_Host, error)
	GetAttachedNetworkStorages(id int, nasType string) ([]datatypes.SoftLayer_Network_Storage, error)
	GetDatacenter(id int) (datatypes.SoftLayer_Location, error)
	GetPrimaryIpAddress(id int) (string, error)
	GetPrimaryBackendIpAddress(id int) (string, error)

	PowerOff(instanceId int) (bool, error)
	PowerOffSoft(instanceId int) (bool, error)
	PowerOn(instanceId int) (bool, error)

	RebootDefault(instanceId int) (bool, error)
	RebootSoft(instanceId int) (bool, error)
	RebootHard(instanceId int) (bool, error)

	SetTags(instanceId int, tags []string) (bool, error)
}
