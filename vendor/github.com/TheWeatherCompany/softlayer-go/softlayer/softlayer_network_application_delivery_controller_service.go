package softlayer

import (
	datatypes "github.com/TheWeatherCompany/softlayer-go/data_types"
)

type NetworkApplicationDeliveryControllerCreateOptions struct {
	Speed    int
	Version  string
	Plan     string
	IpCount  int
	Location string
}

type SoftLayer_Network_Application_Delivery_Controller_Service interface {
	Service

	CreateNetscalerVPX(createOptions *NetworkApplicationDeliveryControllerCreateOptions) (datatypes.SoftLayer_Network_Application_Delivery_Controller, error)
	CreateVirtualIpAddress(nadcId int, template datatypes.SoftLayer_Network_LoadBalancer_VirtualIpAddress_Template) (bool, error)

	CreateLoadBalancerService(vipId string, nadcId int, template []datatypes.SoftLayer_Network_LoadBalancer_Service_Template) (bool, error)

	DeleteVirtualIpAddress(nadcId int, name string) (bool, error)
	DeleteObject(id int) (bool, error)
	DeleteLoadBalancerService(nadcId int, vipId string, serviceId string) (bool, error)

	EditVirtualIpAddress(nadcId int, template datatypes.SoftLayer_Network_LoadBalancer_VirtualIpAddress_Template) (bool, error)

	GetObject(id int) (datatypes.SoftLayer_Network_Application_Delivery_Controller, error)
	GetBillingItem(id int) (datatypes.SoftLayer_Billing_Item, error)
	GetVirtualIpAddress(nadcId int, vipName string) (datatypes.SoftLayer_Network_LoadBalancer_VirtualIpAddress, error)
	GetLoadBalancerService(nadcId int, vipId string, serviceId string) (datatypes.SoftLayer_Network_LoadBalancer_Service, error)

	FindCreatePriceItems(createOptions *NetworkApplicationDeliveryControllerCreateOptions) ([]datatypes.SoftLayer_Product_Item_Price, error)
}
