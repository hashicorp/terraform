package services_test

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	slclientfakes "github.com/TheWeatherCompany/softlayer-go/client/fakes"
	datatypes "github.com/TheWeatherCompany/softlayer-go/data_types"
	softlayer "github.com/TheWeatherCompany/softlayer-go/softlayer"
	testhelpers "github.com/TheWeatherCompany/softlayer-go/test_helpers"
)

var _ = Describe("SoftLayer_Product_Order", func() {
	var (
		username, apiKey string

		fakeClient *slclientfakes.FakeSoftLayerClient

		productOrderService softlayer.SoftLayer_Product_Order_Service
		err                 error
	)

	BeforeEach(func() {
		username = os.Getenv("SL_USERNAME")
		Expect(username).ToNot(Equal(""))

		apiKey = os.Getenv("SL_API_KEY")
		Expect(apiKey).ToNot(Equal(""))

		fakeClient = slclientfakes.NewFakeSoftLayerClient(username, apiKey)
		Expect(fakeClient).ToNot(BeNil())

		productOrderService, err = fakeClient.GetSoftLayer_Product_Order_Service()
		Expect(err).ToNot(HaveOccurred())
		Expect(productOrderService).ToNot(BeNil())
	})

	Context("#GetName", func() {
		It("returns the name for the service", func() {
			name := productOrderService.GetName()
			Expect(name).To(Equal("SoftLayer_Product_Order"))
		})
	})

	Context("#PlaceOrder", func() {
		BeforeEach(func() {
			fakeClient.FakeHttpClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Product_Order_placeOrder.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns an instance of datatypes.SoftLayer_Container_Product_Order_Receipt", func() {
			receipt, err := productOrderService.PlaceOrder(datatypes.SoftLayer_Container_Product_Order{})
			Expect(err).ToNot(HaveOccurred())
			Expect(receipt).ToNot(BeNil())
			Expect(receipt.OrderId).To(Equal(123))
		})

		Context("when HTTP client returns error codes 40x or 50x", func() {
			It("fails for error code 40x", func() {
				errorCodes := []int{400, 401, 499}
				for _, errorCode := range errorCodes {
					fakeClient.FakeHttpClient.DoRawHttpRequestInt = errorCode

					_, err := productOrderService.PlaceOrder(datatypes.SoftLayer_Container_Product_Order{})
					Expect(err).To(HaveOccurred())
				}
			})

			It("fails for error code 50x", func() {
				errorCodes := []int{500, 501, 599}
				for _, errorCode := range errorCodes {
					fakeClient.FakeHttpClient.DoRawHttpRequestInt = errorCode

					_, err := productOrderService.PlaceOrder(datatypes.SoftLayer_Container_Product_Order{})
					Expect(err).To(HaveOccurred())
				}
			})
		})
	})

	Context("#PlaceContainerOrderNetworkPerformanceStorageIscsi", func() {
		BeforeEach(func() {
			fakeClient.FakeHttpClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Product_Order_placeOrder.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns an instance of datatypes.SoftLayer_Container_Product_Order_Receipt", func() {
			receipt, err := productOrderService.PlaceContainerOrderNetworkPerformanceStorageIscsi(datatypes.SoftLayer_Container_Product_Order_Network_PerformanceStorage_Iscsi{})
			Expect(err).ToNot(HaveOccurred())
			Expect(receipt).ToNot(BeNil())
			Expect(receipt.OrderId).To(Equal(123))
		})
		Context("when HTTP client returns error codes 40x or 50x", func() {
			It("fails for error code 40x", func() {
				errorCodes := []int{400, 401, 499}
				for _, errorCode := range errorCodes {
					fakeClient.FakeHttpClient.DoRawHttpRequestInt = errorCode

					_, err := productOrderService.PlaceContainerOrderNetworkPerformanceStorageIscsi(datatypes.SoftLayer_Container_Product_Order_Network_PerformanceStorage_Iscsi{})
					Expect(err).To(HaveOccurred())
				}
			})

			It("fails for error code 50x", func() {
				errorCodes := []int{500, 501, 599}
				for _, errorCode := range errorCodes {
					fakeClient.FakeHttpClient.DoRawHttpRequestInt = errorCode

					_, err := productOrderService.PlaceContainerOrderNetworkPerformanceStorageIscsi(datatypes.SoftLayer_Container_Product_Order_Network_PerformanceStorage_Iscsi{})
					Expect(err).To(HaveOccurred())
				}
			})
		})
	})

	Context("#PlaceContainerOrderVirtualGuestUpgrade", func() {
		BeforeEach(func() {
			fakeClient.FakeHttpClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Product_Order_placeOrder.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns an instance of datatypes.SoftLayer_Container_Product_Order_Receipt", func() {
			receipt, err := productOrderService.PlaceContainerOrderVirtualGuestUpgrade(datatypes.SoftLayer_Container_Product_Order_Virtual_Guest_Upgrade{})
			Expect(err).ToNot(HaveOccurred())
			Expect(receipt).ToNot(BeNil())
			Expect(receipt.OrderId).To(Equal(123))
		})

		Context("when HTTP client returns error codes 40x or 50x", func() {
			It("fails for error code 40x", func() {
				errorCodes := []int{400, 401, 499}
				for _, errorCode := range errorCodes {
					fakeClient.FakeHttpClient.DoRawHttpRequestInt = errorCode

					_, err := productOrderService.PlaceContainerOrderVirtualGuestUpgrade(datatypes.SoftLayer_Container_Product_Order_Virtual_Guest_Upgrade{})
					Expect(err).To(HaveOccurred())
				}
			})

			It("fails for error code 50x", func() {
				errorCodes := []int{500, 501, 599}
				for _, errorCode := range errorCodes {
					fakeClient.FakeHttpClient.DoRawHttpRequestInt = errorCode

					_, err := productOrderService.PlaceContainerOrderVirtualGuestUpgrade(datatypes.SoftLayer_Container_Product_Order_Virtual_Guest_Upgrade{})
					Expect(err).To(HaveOccurred())
				}
			})
		})
	})
})
