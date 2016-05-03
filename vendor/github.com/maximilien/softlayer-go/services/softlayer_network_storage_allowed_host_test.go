package services_test

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	slclientfakes "github.com/maximilien/softlayer-go/client/fakes"
	softlayer "github.com/maximilien/softlayer-go/softlayer"
	testhelpers "github.com/maximilien/softlayer-go/test_helpers"
)

var _ = Describe("SoftLayer_Network_Storage_Allowed_Host", func() {
	var (
		username, apiKey string

		fakeClient *slclientfakes.FakeSoftLayerClient

		networkStorageAllowedHostService softlayer.SoftLayer_Network_Storage_Allowed_Host_Service
		err                              error
	)

	BeforeEach(func() {
		username = os.Getenv("SL_USERNAME")
		Expect(username).ToNot(Equal(""))

		apiKey = os.Getenv("SL_API_KEY")
		Expect(apiKey).ToNot(Equal(""))

		fakeClient = slclientfakes.NewFakeSoftLayerClient(username, apiKey)
		Expect(fakeClient).ToNot(BeNil())

		networkStorageAllowedHostService, err = fakeClient.GetSoftLayer_Network_Storage_Allowed_Host_Service()
		Expect(err).ToNot(HaveOccurred())
		Expect(networkStorageAllowedHostService).ToNot(BeNil())
	})

	Context("#GetName", func() {
		It("returns the name for the service", func() {
			name := networkStorageAllowedHostService.GetName()
			Expect(name).To(Equal("SoftLayer_Network_Storage_Allowed_Host"))
		})
	})

	Context("#GetCredential", func() {
		BeforeEach(func() {
			fakeClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Network_Storage_Allowed_Host_Service_getCredential.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("return the credential with allowed host id", func() {
			credential, err := networkStorageAllowedHostService.GetCredential(123456)
			Expect(err).NotTo(HaveOccurred())
			Expect(credential).ToNot(BeNil())
			Expect(credential.Username).To(Equal("fake-username"))
			Expect(credential.Password).To(Equal("fake-password"))
		})
	})
})
