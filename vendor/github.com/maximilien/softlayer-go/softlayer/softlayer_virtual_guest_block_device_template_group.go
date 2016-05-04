package softlayer

import (
	datatypes "github.com/maximilien/softlayer-go/data_types"
)

type SoftLayer_Virtual_Guest_Block_Device_Template_Group_Service interface {
	Service

	AddLocations(id int, locations []datatypes.SoftLayer_Location) (bool, error)

	CreateFromExternalSource(configuration datatypes.SoftLayer_Container_Virtual_Guest_Block_Device_Template_Configuration) (datatypes.SoftLayer_Virtual_Guest_Block_Device_Template_Group, error)
	CreatePublicArchiveTransaction(id int, groupName string, summary string, note string, locations []datatypes.SoftLayer_Location) (int, error)
	CopyToExternalSource(configuration datatypes.SoftLayer_Container_Virtual_Guest_Block_Device_Template_Configuration) (bool, error)

	DeleteObject(id int) (datatypes.SoftLayer_Provisioning_Version1_Transaction, error)
	DenySharingAccess(id int, accountId int) (bool, error)

	GetObject(id int) (datatypes.SoftLayer_Virtual_Guest_Block_Device_Template_Group, error)
	GetDatacenters(id int) ([]datatypes.SoftLayer_Location, error)
	GetSshKeys(id int) ([]datatypes.SoftLayer_Security_Ssh_Key, error)
	GetStatus(id int) (datatypes.SoftLayer_Virtual_Guest_Block_Device_Template_Group_Status, error)

	GetStorageLocations(id int) ([]datatypes.SoftLayer_Location, error)

	GetImageType(id int) (datatypes.SoftLayer_Image_Type, error)
	GetImageTypeKeyName(id int) (string, error)

	GetTransaction(id int) (datatypes.SoftLayer_Provisioning_Version1_Transaction, error)

	PermitSharingAccess(id int, accountId int) (bool, error)

	RemoveLocations(id int, locations []datatypes.SoftLayer_Location) (bool, error)

	SetAvailableLocations(id int, locations []datatypes.SoftLayer_Location) (bool, error)
}
