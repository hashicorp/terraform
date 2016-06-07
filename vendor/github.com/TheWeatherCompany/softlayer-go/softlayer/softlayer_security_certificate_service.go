package softlayer

import (
	datatypes "github.com/TheWeatherCompany/softlayer-go/data_types"
)

type SoftLayer_Security_Certificate_Service interface {
	Service

	CreateSecurityCertificate(template datatypes.SoftLayer_Security_Certificate_Template) (datatypes.SoftLayer_Security_Certificate, error)
	DeleteObject(id int) (bool, error)
	GetObject(id int) (datatypes.SoftLayer_Security_Certificate, error)
}
