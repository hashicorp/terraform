package softlayer

import (
	"log"

	slclient "github.com/TheWeatherCompany/softlayer-go/client"
	softlayer "github.com/TheWeatherCompany/softlayer-go/softlayer"
)

type Config struct {
	Username string
	ApiKey   string
}

type Client struct {
	virtualGuestService                         softlayer.SoftLayer_Virtual_Guest_Service
	sshKeyService                               softlayer.SoftLayer_Security_Ssh_Key_Service
	productOrderService                         softlayer.SoftLayer_Product_Order_Service
	dnsDomainResourceRecordService              softlayer.SoftLayer_Dns_Domain_ResourceRecord_Service
	dnsDomainService                            softlayer.SoftLayer_Dns_Domain_Service
	networkApplicationDeliveryControllerService softlayer.SoftLayer_Network_Application_Delivery_Controller_Service
}

func (c *Config) Client() (*Client, error) {
	slc := slclient.NewSoftLayerClient(c.Username, c.ApiKey)
	virtualGuestService, err := slc.GetSoftLayer_Virtual_Guest_Service()

	if err != nil {
		return nil, err
	}

	networkApplicationDeliveryControllerService, err := slc.GetSoftLayer_Network_Application_Delivery_Controller_Service()

	if err != nil {
		return nil, err
	}

	sshKeyService, err := slc.GetSoftLayer_Security_Ssh_Key_Service()

	if err != nil {
		return nil, err
	}

	dnsDomainService, err := slc.GetSoftLayer_Dns_Domain_Service()

	if err != nil {
		return nil, err
	}

	dnsDomainResourceRecordService, err := slc.GetSoftLayer_Dns_Domain_ResourceRecord_Service()

	if err != nil {
		return nil, err
	}

	client := &Client{
		virtualGuestService:                         virtualGuestService,
		sshKeyService:                               sshKeyService,
		dnsDomainService:                            dnsDomainService,
		dnsDomainResourceRecordService:              dnsDomainResourceRecordService,
		networkApplicationDeliveryControllerService: networkApplicationDeliveryControllerService,
	}

	log.Println("[INFO] Created SoftLayer client")

	return client, nil
}
