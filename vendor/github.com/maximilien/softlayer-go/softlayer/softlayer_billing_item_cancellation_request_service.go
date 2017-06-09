package softlayer

import (
	datatypes "github.com/maximilien/softlayer-go/data_types"
)

type SoftLayer_Billing_Item_Cancellation_Request_Service interface {
	Service

	CreateObject(request datatypes.SoftLayer_Billing_Item_Cancellation_Request) (datatypes.SoftLayer_Billing_Item_Cancellation_Request, error)
}
