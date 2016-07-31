package softlayer

import (
	"log"

	slclient "github.com/maximilien/softlayer-go/client"
	softlayer "github.com/maximilien/softlayer-go/softlayer"
)

type Config struct {
	Username string
	ApiKey   string
}

type Client struct {
	virtualGuestService softlayer.SoftLayer_Virtual_Guest_Service
	sshKeyService       softlayer.SoftLayer_Security_Ssh_Key_Service
	productOrderService softlayer.SoftLayer_Product_Order_Service
}

func (c *Config) Client() (*Client, error) {
	slc := slclient.NewSoftLayerClient(c.Username, c.ApiKey)
	virtualGuestService, err := slc.GetSoftLayer_Virtual_Guest_Service()

	if err != nil {
		return nil, err
	}

	sshKeyService, err := slc.GetSoftLayer_Security_Ssh_Key_Service()

	client := &Client{
		virtualGuestService: virtualGuestService,
		sshKeyService:       sshKeyService,
	}

	log.Println("[INFO] Created SoftLayer client")

	return client, nil
}
