package softlayer

import (
	datatypes "github.com/maximilien/softlayer-go/data_types"
)

type SoftLayer_Network_Storage_Allowed_Host_Service interface {
	Service

	GetCredential(allowedHostId int) (datatypes.SoftLayer_Network_Storage_Credential, error)
}
