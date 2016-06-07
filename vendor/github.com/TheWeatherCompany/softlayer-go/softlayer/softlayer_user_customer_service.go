package softlayer

import (
	"github.com/TheWeatherCompany/softlayer-go/data_types"
)

type SoftLayer_User_Customer_Service interface {
	Service

	AddApiAuthenticationKey(userId int) error
	GetApiAuthenticationKeys(userId int) ([]data_types.SoftLayer_User_Customer_ApiAuthentication, error)
}
