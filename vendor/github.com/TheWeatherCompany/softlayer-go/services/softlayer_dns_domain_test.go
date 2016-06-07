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

var _ = Describe("SoftLayer_Dns_Domain", func() {
	var (
		username, apiKey string

		fakeClient *slclientfakes.FakeSoftLayerClient

		dnsDomainService softlayer.SoftLayer_Dns_Domain_Service
		err              error
	)

	BeforeEach(func() {
		username = os.Getenv("SL_USERNAME")
		Expect(username).NotTo(Equal(""))

		apiKey = os.Getenv("SL_API_KEY")
		Expect(apiKey).NotTo(Equal(""))

		fakeClient = slclientfakes.NewFakeSoftLayerClient(username, apiKey)
		Expect(fakeClient).NotTo(BeNil())

		dnsDomainService, err = fakeClient.GetSoftLayer_Dns_Domain_Service()
		Expect(err).ToNot(HaveOccurred())
		Expect(dnsDomainService).NotTo(BeNil())
	})

	Context("#GetName", func() {
		It("returns the name for the service", func() {
			name := dnsDomainService.GetName()
			Expect(name).To(Equal("SoftLayer_Dns_Domain"))
		})
	})

	Context("#CreateDns", func() {
		BeforeEach(func() {
			fakeClient.FakeHttpClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Dns_Domain_createObject.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns an instance of datatypes.SoftLayer_Dns_Domain", func() {
			dns, err := dnsDomainService.CreateObject(datatypes.SoftLayer_Dns_Domain_Template{})
			Expect(err).ToNot(HaveOccurred())
			Expect(dns).NotTo(BeNil())
			Expect(dns.Id).NotTo(BeNil())
			Expect(dns.Serial).NotTo(BeNil())
			Expect(dns.UpdateDate).NotTo(BeNil())
			Expect(dns.Name).To(Equal("qwerty123ff.com"))
		})

		Context("when HTTP client returns error codes 40x or 50x", func() {
			It("fails for error code 40x", func() {
				errorCodes := []int{400, 401, 499}
				for _, errorCode := range errorCodes {
					fakeClient.FakeHttpClient.DoRawHttpRequestInt = errorCode

					_, err := dnsDomainService.CreateObject(datatypes.SoftLayer_Dns_Domain_Template{})
					Expect(err).To(HaveOccurred())
				}
			})

			It("fails for error code 50x", func() {
				errorCodes := []int{500, 501, 599}
				for _, errorCode := range errorCodes {
					fakeClient.FakeHttpClient.DoRawHttpRequestInt = errorCode

					_, err := dnsDomainService.CreateObject(datatypes.SoftLayer_Dns_Domain_Template{})
					Expect(err).To(HaveOccurred())
				}
			})
		})
	})
})
