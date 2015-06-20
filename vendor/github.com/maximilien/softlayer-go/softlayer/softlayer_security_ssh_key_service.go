package softlayer

import (
	datatypes "github.com/maximilien/softlayer-go/data_types"
)

type SoftLayer_Security_Ssh_Key_Service interface {
	Service

	CreateObject(template datatypes.SoftLayer_Security_Ssh_Key) (datatypes.SoftLayer_Security_Ssh_Key, error)
	GetObject(sshkeyId int) (datatypes.SoftLayer_Security_Ssh_Key, error)
	EditObject(sshkeyId int, template datatypes.SoftLayer_Security_Ssh_Key) (bool, error)
	DeleteObject(sshKeyId int) (bool, error)

	GetSoftwarePasswords(sshKeyId int) ([]datatypes.SoftLayer_Software_Component_Password, error)
}
