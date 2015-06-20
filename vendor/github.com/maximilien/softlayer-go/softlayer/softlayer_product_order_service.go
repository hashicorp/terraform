package softlayer

import (
	datatypes "github.com/maximilien/softlayer-go/data_types"
)

type SoftLayer_Product_Order_Service interface {
	Service

	PlaceOrder(order datatypes.SoftLayer_Container_Product_Order) (datatypes.SoftLayer_Container_Product_Order_Receipt, error)
	PlaceContainerOrderNetworkPerformanceStorageIscsi(order datatypes.SoftLayer_Container_Product_Order_Network_PerformanceStorage_Iscsi) (datatypes.SoftLayer_Container_Product_Order_Receipt, error)
	PlaceContainerOrderVirtualGuestUpgrade(order datatypes.SoftLayer_Container_Product_Order_Virtual_Guest_Upgrade) (datatypes.SoftLayer_Container_Product_Order_Receipt, error)
}
