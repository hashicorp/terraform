package softlayer

import (
	datatypes "github.com/maximilien/softlayer-go/data_types"
)

// Modifying existing SoftLayer_Dns_Domain entries is not possible. Changes to zone names should be refactored to creation of new zones.
// https://sldn.softlayer.com/blog/phil/Getting-started-DNS
type SoftLayer_Dns_Domain_Service interface {
	Service

	CreateObject(template datatypes.SoftLayer_Dns_Domain_Template) (datatypes.SoftLayer_Dns_Domain, error)
	DeleteObject(dnsId int) (bool, error)
	GetObject(dnsId int) (datatypes.SoftLayer_Dns_Domain, error)
}
