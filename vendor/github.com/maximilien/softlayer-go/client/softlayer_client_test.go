package client_test

import (
	"bytes"
	"errors"
	"net"
	"net/http"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	slclient "github.com/maximilien/softlayer-go/client"
	softlayer "github.com/maximilien/softlayer-go/softlayer"
)

var _ = Describe("SoftLayerClient", func() {
	var (
		username string
		apiKey   string

		client softlayer.Client
	)

	BeforeEach(func() {
		username = os.Getenv("SL_USERNAME")
		apiKey = os.Getenv("SL_API_KEY")

		client = slclient.NewSoftLayerClient(username, apiKey)
	})

	Context("#NewSoftLayerClient", func() {
		It("creates a new client with username and apiKey", func() {
			Expect(username).ToNot(Equal(""), "username cannot be empty, set SL_USERNAME")
			Expect(apiKey).ToNot(Equal(""), "apiKey cannot be empty, set SL_API_KEY")

			client = slclient.NewSoftLayerClient(username, apiKey)
			Expect(client).ToNot(BeNil())
		})
	})

	Context("#NewSoftLayerClient_HTTPClient", func() {
		It("creates a new client which should have an initialized default HTTP client", func() {
			client = slclient.NewSoftLayerClient(username, apiKey)
			if c, ok := client.(*slclient.SoftLayerClient); ok {
				Expect(c.HTTPClient).ToNot(BeNil())
			}
		})

		It("creates a new client with a custom HTTP client", func() {
			client = slclient.NewSoftLayerClient(username, apiKey)
			c, _ := client.(*slclient.SoftLayerClient)

			// Assign a malformed dialer to test if the HTTP client really works
			var errDialFailed = errors.New("dial failed")
			c.HTTPClient = &http.Client{
				Transport: &http.Transport{
					Dial: func(network, addr string) (net.Conn, error) {
						return nil, errDialFailed
					},
				},
			}

			_, err := c.DoRawHttpRequest("/foo", "application/text", bytes.NewBufferString("random text"))
			Expect(err).To(Equal(errDialFailed))

		})
	})

	Context("#GetService", func() {
		It("returns a service with name specified", func() {
			accountService, err := client.GetService("SoftLayer_Account")
			Expect(err).ToNot(HaveOccurred())
			Expect(accountService).ToNot(BeNil())
		})

		It("fails when passed a bad service name", func() {
			_, err := client.GetService("fake-service-name")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("softlayer-go does not support service 'fake-service-name'"))
		})
	})

	Context("#GetSoftLayer_Account_Service", func() {
		It("returns a instance implemementing the SoftLayer_Account_Service interface", func() {
			var accountService softlayer.SoftLayer_Account_Service
			accountService, err := client.GetSoftLayer_Account_Service()
			Expect(err).ToNot(HaveOccurred())
			Expect(accountService).ToNot(BeNil())
		})
	})

	Context("#GetSoftLayer_Virtual_Guest_Service", func() {
		It("returns a instance implemementing the SoftLayer_Virtual_Guest_Service interface", func() {
			var virtualGuestService softlayer.SoftLayer_Virtual_Guest_Service
			virtualGuestService, err := client.GetSoftLayer_Virtual_Guest_Service()
			Expect(err).ToNot(HaveOccurred())
			Expect(virtualGuestService).ToNot(BeNil())
		})
	})

	Context("#GetSoftLayer_Ssh_Key_Service", func() {
		It("returns a instance implemementing the SoftLayer_Ssh_Key_Service interface", func() {
			var sshKeyService softlayer.SoftLayer_Security_Ssh_Key_Service
			sshKeyService, err := client.GetSoftLayer_Security_Ssh_Key_Service()
			Expect(err).ToNot(HaveOccurred())
			Expect(sshKeyService).ToNot(BeNil())
		})
	})

	Context("#GetSoftLayer_Product_Order_Service", func() {
		It("returns a instance implemementing the SoftLayer_Product_Order_Service interface", func() {
			var productOrderService softlayer.SoftLayer_Product_Order_Service
			productOrderService, err := client.GetSoftLayer_Product_Order_Service()
			Expect(err).ToNot(HaveOccurred())
			Expect(productOrderService).ToNot(BeNil())
		})
	})

	Context("#GetSoftLayer_Product_Package_Service", func() {
		It("returns a instance implemementing the SoftLayer_Product_Package interface", func() {
			var productPackageService softlayer.SoftLayer_Product_Package_Service
			productPackageService, err := client.GetSoftLayer_Product_Package_Service()
			Expect(err).ToNot(HaveOccurred())
			Expect(productPackageService).ToNot(BeNil())
		})
	})

	Context("#GetSoftLayer_Network_Storage_Service", func() {
		It("returns a instance implemementing the GetSoftLayer_Network_Storage_Service interface", func() {
			var networkStorageService softlayer.SoftLayer_Network_Storage_Service
			networkStorageService, err := client.GetSoftLayer_Network_Storage_Service()
			Expect(err).ToNot(HaveOccurred())
			Expect(networkStorageService).ToNot(BeNil())
		})
	})

	Context("#GetSoftLayer_Network_Storage_Allowed_Host_Service", func() {
		It("returns a instance implemementing the GetSoftLayer_Network_Storage_Allowed_Host_Service interface", func() {
			var networkStorageAllowedHostService softlayer.SoftLayer_Network_Storage_Allowed_Host_Service
			networkStorageAllowedHostService, err := client.GetSoftLayer_Network_Storage_Allowed_Host_Service()
			Expect(err).ToNot(HaveOccurred())
			Expect(networkStorageAllowedHostService).ToNot(BeNil())
		})
	})

	Context("#GetSoftLayer_Billing_Item_Cancellation_Request", func() {
		It("returns a instance implemementing the SoftLayer_Billing_Item_Cancellation_Request interface", func() {
			var billingItemCancellationRequestService softlayer.SoftLayer_Billing_Item_Cancellation_Request_Service
			billingItemCancellationRequestService, err := client.GetSoftLayer_Billing_Item_Cancellation_Request_Service()
			Expect(err).ToNot(HaveOccurred())
			Expect(billingItemCancellationRequestService).ToNot(BeNil())
		})
	})

	Context("#GetSoftLayer_Virtual_Guest_Block_Device_Template_Group_Service", func() {
		It("returns a instance implemementing the SoftLayer_Virtual_Guest_Block_Device_Template_Group_Service interface", func() {
			var vgbdtgService softlayer.SoftLayer_Virtual_Guest_Block_Device_Template_Group_Service
			vgbdtgService, err := client.GetSoftLayer_Virtual_Guest_Block_Device_Template_Group_Service()
			Expect(err).ToNot(HaveOccurred())
			Expect(vgbdtgService).ToNot(BeNil())
		})
	})

	Context("#GetSoftLayer_Hardware_Service", func() {
		It("returns an instance implemementing the SoftLayer_Hardware_Service interface", func() {
			var hardwareService softlayer.SoftLayer_Hardware_Service
			hardwareService, err := client.GetSoftLayer_Hardware_Service()
			Expect(err).ToNot(HaveOccurred())
			Expect(hardwareService).ToNot(BeNil())
		})
	})
})
