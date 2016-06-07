package services_test

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	slclientfakes "github.com/TheWeatherCompany/softlayer-go/client/fakes"
	softlayer "github.com/TheWeatherCompany/softlayer-go/softlayer"
)

var _ = Describe("SoftLayer_Billing_Item", func() {
	var (
		username, apiKey string

		fakeClient *slclientfakes.FakeSoftLayerClient

		billingItemService softlayer.SoftLayer_Billing_Item_Service
		err                error
	)

	BeforeEach(func() {
		username = os.Getenv("SL_USERNAME")
		Expect(username).ToNot(Equal(""))

		apiKey = os.Getenv("SL_API_KEY")
		Expect(apiKey).ToNot(Equal(""))

		fakeClient = slclientfakes.NewFakeSoftLayerClient(username, apiKey)
		Expect(fakeClient).ToNot(BeNil())

		billingItemService, err = fakeClient.GetSoftLayer_Billing_Item_Service()
		Expect(err).ToNot(HaveOccurred())
		Expect(billingItemService).ToNot(BeNil())
	})

	Context("#GetName", func() {
		It("returns the name for the service", func() {
			name := billingItemService.GetName()
			Expect(name).To(Equal("SoftLayer_Billing_Item"))
		})
	})

	Context("#CancelService", func() {

		It("returns true", func() {
			fakeClient.FakeHttpClient.DoRawHttpRequestResponse = []byte("true")
			deleted, err := billingItemService.CancelService(1234567)
			Expect(err).ToNot(HaveOccurred())
			Expect(deleted).To(BeTrue())
		})

		It("returns false", func() {
			fakeClient.FakeHttpClient.DoRawHttpRequestResponse = []byte("false")
			deleted, err := billingItemService.CancelService(1234567)
			Expect(err).ToNot(HaveOccurred())
			Expect(deleted).To(BeFalse())
		})
	})
})
