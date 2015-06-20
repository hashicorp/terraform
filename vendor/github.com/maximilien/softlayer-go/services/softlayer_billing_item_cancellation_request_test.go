package services_test

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	slclientfakes "github.com/maximilien/softlayer-go/client/fakes"
	datatypes "github.com/maximilien/softlayer-go/data_types"
	softlayer "github.com/maximilien/softlayer-go/softlayer"
	testhelpers "github.com/maximilien/softlayer-go/test_helpers"
)

var _ = Describe("SoftLayer_Billing_Item_Cancellation_Request", func() {
	var (
		username, apiKey string

		fakeClient *slclientfakes.FakeSoftLayerClient

		billingItemCancellationRequestService softlayer.SoftLayer_Billing_Item_Cancellation_Request_Service
		err                                   error
	)

	BeforeEach(func() {
		username = os.Getenv("SL_USERNAME")
		Expect(username).ToNot(Equal(""))

		apiKey = os.Getenv("SL_API_KEY")
		Expect(apiKey).ToNot(Equal(""))

		fakeClient = slclientfakes.NewFakeSoftLayerClient(username, apiKey)
		Expect(fakeClient).ToNot(BeNil())

		billingItemCancellationRequestService, err = fakeClient.GetSoftLayer_Billing_Item_Cancellation_Request_Service()
		Expect(err).ToNot(HaveOccurred())
		Expect(billingItemCancellationRequestService).ToNot(BeNil())
	})

	Context("#GetNam", func() {
		It("returns the name for the service", func() {
			name := billingItemCancellationRequestService.GetName()
			Expect(name).To(Equal("SoftLayer_Billing_Item_Cancellation_Request"))
		})
	})

	Context("#CreateObject", func() {
		BeforeEach(func() {
			fakeClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Billing_Item_Cancellation_Request_Service_createObject.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns an instance of datatypes.SoftLayer_Network_Storage", func() {

			request := datatypes.SoftLayer_Billing_Item_Cancellation_Request{}

			result, err := billingItemCancellationRequestService.CreateObject(request)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Id).To(Equal(123))
			Expect(result.AccountId).To(Equal(456))
			Expect(result.TicketId).To(Equal(789))
		})
	})
})
