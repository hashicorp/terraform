package client

import (
	"errors"
	"fmt"
	"os"

	"github.com/maximilien/softlayer-go/services"
	"github.com/maximilien/softlayer-go/softlayer"
)

const (
	TEMPLATE_ROOT_PATH = "templates"
)

var sl_endpoint_servers = [4]string{"api.softlayer.com", "api.service.softlayer.com", "66.228.119.120", "10.0.80.88"}

type SoftLayerClient struct {
	HttpClient softlayer.HttpClient

	softLayerServices map[string]softlayer.Service
}

func NewSoftLayerClient(username, apiKey string) *SoftLayerClient {
	slc := &SoftLayerClient{
		HttpClient: NewHttpsClient(username, apiKey, GetSLApiEndpoint(), TEMPLATE_ROOT_PATH),

		softLayerServices: map[string]softlayer.Service{},
	}

	slc.initSoftLayerServices()

	return slc
}

//softlayer.Client interface methods

func (slc *SoftLayerClient) GetHttpClient() softlayer.HttpClient {
	return slc.HttpClient
}

func (slc *SoftLayerClient) GetService(serviceName string) (softlayer.Service, error) {
	slService, ok := slc.softLayerServices[serviceName]
	if !ok {
		return nil, errors.New(fmt.Sprintf("softlayer-go does not support service '%s'", serviceName))
	}

	return slService, nil
}

func (slc *SoftLayerClient) GetSoftLayer_Account_Service() (softlayer.SoftLayer_Account_Service, error) {
	slService, err := slc.GetService("SoftLayer_Account")
	if err != nil {
		return nil, err
	}

	return slService.(softlayer.SoftLayer_Account_Service), nil
}

func (slc *SoftLayerClient) GetSoftLayer_Virtual_Guest_Service() (softlayer.SoftLayer_Virtual_Guest_Service, error) {
	slService, err := slc.GetService("SoftLayer_Virtual_Guest")
	if err != nil {
		return nil, err
	}

	return slService.(softlayer.SoftLayer_Virtual_Guest_Service), nil
}

func (slc *SoftLayerClient) GetSoftLayer_Dns_Domain_Service() (softlayer.SoftLayer_Dns_Domain_Service, error) {
	slService, err := slc.GetService("SoftLayer_Dns_Domain")
	if err != nil {
		return nil, err
	}

	return slService.(softlayer.SoftLayer_Dns_Domain_Service), nil
}

func (slc *SoftLayerClient) GetSoftLayer_Virtual_Disk_Image_Service() (softlayer.SoftLayer_Virtual_Disk_Image_Service, error) {
	slService, err := slc.GetService("SoftLayer_Virtual_Disk_Image")
	if err != nil {
		return nil, err
	}

	return slService.(softlayer.SoftLayer_Virtual_Disk_Image_Service), nil
}

func (slc *SoftLayerClient) GetSoftLayer_Security_Ssh_Key_Service() (softlayer.SoftLayer_Security_Ssh_Key_Service, error) {
	slService, err := slc.GetService("SoftLayer_Security_Ssh_Key")
	if err != nil {
		return nil, err
	}

	return slService.(softlayer.SoftLayer_Security_Ssh_Key_Service), nil
}

func (slc *SoftLayerClient) GetSoftLayer_Product_Package_Service() (softlayer.SoftLayer_Product_Package_Service, error) {
	slService, err := slc.GetService("SoftLayer_Product_Package")
	if err != nil {
		return nil, err
	}

	return slService.(softlayer.SoftLayer_Product_Package_Service), nil
}

func (slc *SoftLayerClient) GetSoftLayer_Virtual_Guest_Block_Device_Template_Group_Service() (softlayer.SoftLayer_Virtual_Guest_Block_Device_Template_Group_Service, error) {
	slService, err := slc.GetService("SoftLayer_Virtual_Guest_Block_Device_Template_Group")
	if err != nil {
		return nil, err
	}

	return slService.(softlayer.SoftLayer_Virtual_Guest_Block_Device_Template_Group_Service), nil
}

func (slc *SoftLayerClient) GetSoftLayer_Network_Storage_Service() (softlayer.SoftLayer_Network_Storage_Service, error) {
	slService, err := slc.GetService("SoftLayer_Network_Storage")
	if err != nil {
		return nil, err
	}

	return slService.(softlayer.SoftLayer_Network_Storage_Service), nil
}

func (slc *SoftLayerClient) GetSoftLayer_Network_Storage_Allowed_Host_Service() (softlayer.SoftLayer_Network_Storage_Allowed_Host_Service, error) {
	slService, err := slc.GetService("SoftLayer_Network_Storage_Allowed_Host")
	if err != nil {
		return nil, err
	}

	return slService.(softlayer.SoftLayer_Network_Storage_Allowed_Host_Service), nil
}

func (slc *SoftLayerClient) GetSoftLayer_Product_Order_Service() (softlayer.SoftLayer_Product_Order_Service, error) {
	slService, err := slc.GetService("SoftLayer_Product_Order")
	if err != nil {
		return nil, err
	}

	return slService.(softlayer.SoftLayer_Product_Order_Service), nil
}

func (slc *SoftLayerClient) GetSoftLayer_Billing_Item_Cancellation_Request_Service() (softlayer.SoftLayer_Billing_Item_Cancellation_Request_Service, error) {
	slService, err := slc.GetService("SoftLayer_Billing_Item_Cancellation_Request")
	if err != nil {
		return nil, err
	}

	return slService.(softlayer.SoftLayer_Billing_Item_Cancellation_Request_Service), nil
}

func (slc *SoftLayerClient) GetSoftLayer_Billing_Item_Service() (softlayer.SoftLayer_Billing_Item_Service, error) {
	slService, err := slc.GetService("SoftLayer_Billing_Item")
	if err != nil {
		return nil, err
	}

	return slService.(softlayer.SoftLayer_Billing_Item_Service), nil
}

func (slc *SoftLayerClient) GetSoftLayer_Hardware_Service() (softlayer.SoftLayer_Hardware_Service, error) {
	slService, err := slc.GetService("SoftLayer_Hardware")
	if err != nil {
		return nil, err
	}

	return slService.(softlayer.SoftLayer_Hardware_Service), nil
}

func (slc *SoftLayerClient) GetSoftLayer_Dns_Domain_ResourceRecord_Service() (softlayer.SoftLayer_Dns_Domain_ResourceRecord_Service, error) {
	slService, err := slc.GetService("SoftLayer_Dns_Domain_ResourceRecord")
	if err != nil {
		return nil, err
	}

	return slService.(softlayer.SoftLayer_Dns_Domain_ResourceRecord_Service), nil
}

func GetSLApiEndpoint() string {
	sl_api_endpoint := os.Getenv("SL_API_ENDPOINT")
	var included bool = false

	for _, server := range sl_endpoint_servers {
		if server == sl_api_endpoint {
			included = true
			break
		}
	}
	if !included {
		sl_api_endpoint = "api.softlayer.com"
	}
	softlayer_api_url := fmt.Sprintf("%s/rest/v3", sl_api_endpoint)

	return softlayer_api_url
}

//Private methods

func (slc *SoftLayerClient) initSoftLayerServices() {
	slc.softLayerServices["SoftLayer_Account"] = services.NewSoftLayer_Account_Service(slc)
	slc.softLayerServices["SoftLayer_Virtual_Guest"] = services.NewSoftLayer_Virtual_Guest_Service(slc)
	slc.softLayerServices["SoftLayer_Virtual_Disk_Image"] = services.NewSoftLayer_Virtual_Disk_Image_Service(slc)
	slc.softLayerServices["SoftLayer_Security_Ssh_Key"] = services.NewSoftLayer_Security_Ssh_Key_Service(slc)
	slc.softLayerServices["SoftLayer_Product_Package"] = services.NewSoftLayer_Product_Package_Service(slc)
	slc.softLayerServices["SoftLayer_Network_Storage"] = services.NewSoftLayer_Network_Storage_Service(slc)
	slc.softLayerServices["SoftLayer_Network_Storage_Allowed_Host"] = services.NewSoftLayer_Network_Storage_Allowed_Host_Service(slc)
	slc.softLayerServices["SoftLayer_Product_Order"] = services.NewSoftLayer_Product_Order_Service(slc)
	slc.softLayerServices["SoftLayer_Billing_Item_Cancellation_Request"] = services.NewSoftLayer_Billing_Item_Cancellation_Request_Service(slc)
	slc.softLayerServices["SoftLayer_Billing_Item"] = services.NewSoftLayer_Billing_Item_Service(slc)
	slc.softLayerServices["SoftLayer_Virtual_Guest_Block_Device_Template_Group"] = services.NewSoftLayer_Virtual_Guest_Block_Device_Template_Group_Service(slc)
	slc.softLayerServices["SoftLayer_Hardware"] = services.NewSoftLayer_Hardware_Service(slc)
	slc.softLayerServices["SoftLayer_Dns_Domain"] = services.NewSoftLayer_Dns_Domain_Service(slc)
	slc.softLayerServices["SoftLayer_Dns_Domain_ResourceRecord"] = services.NewSoftLayer_Dns_Domain_ResourceRecord_Service(slc)
}
