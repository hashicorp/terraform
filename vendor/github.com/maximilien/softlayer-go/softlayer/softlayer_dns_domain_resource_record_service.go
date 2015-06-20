package softlayer

import (
	datatypes "github.com/maximilien/softlayer-go/data_types"
)

type SoftLayer_Dns_Domain_ResourceRecord_Service interface {
	Service

	CreateObject(template datatypes.SoftLayer_Dns_Domain_ResourceRecord_Template) (datatypes.SoftLayer_Dns_Domain_ResourceRecord, error)
	GetObject(recordId int) (datatypes.SoftLayer_Dns_Domain_ResourceRecord, error)
	DeleteObject(recordId int) (bool, error)
	EditObject(recordId int, template datatypes.SoftLayer_Dns_Domain_ResourceRecord) (bool, error)
}
