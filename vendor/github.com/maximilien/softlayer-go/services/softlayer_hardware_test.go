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

var _ = Describe("SoftLayer_Hardware", func() {
	var (
		username, apiKey string

		fakeClient *slclientfakes.FakeSoftLayerClient

		hardwareService softlayer.SoftLayer_Hardware_Service
		err             error
	)

	BeforeEach(func() {
		username = os.Getenv("SL_USERNAME")
		Expect(username).ToNot(Equal(""))

		apiKey = os.Getenv("SL_API_KEY")
		Expect(apiKey).ToNot(Equal(""))

		fakeClient = slclientfakes.NewFakeSoftLayerClient(username, apiKey)
		Expect(fakeClient).ToNot(BeNil())

		hardwareService, err = fakeClient.GetSoftLayer_Hardware_Service()
		Expect(err).ToNot(HaveOccurred())
		Expect(hardwareService).ToNot(BeNil())
	})

	Context("#GetName", func() {
		It("returns the name for the service", func() {
			name := hardwareService.GetName()
			Expect(name).To(Equal("SoftLayer_Hardware"))
		})
	})

	Context("#CreateObject", func() {
		BeforeEach(func() {
			fakeClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Hardware_Service_createObject.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("creates a new SoftLayer_Virtual_Guest instance", func() {
			template := datatypes.SoftLayer_Hardware_Template{
				Hostname:                     "softlayer",
				Domain:                       "testing.com",
				ProcessorCoreAmount:          2,
				MemoryCapacity:               2,
				HourlyBillingFlag:            true,
				OperatingSystemReferenceCode: "UBUNTU_LATEST",
				Datacenter: &datatypes.Datacenter{
					Name: "ams01",
				},
			}

			hardware, err := hardwareService.CreateObject(template)
			Expect(err).ToNot(HaveOccurred())
			Expect(hardware.Id).To(Equal(123))
			Expect(hardware.Hostname).To(Equal("softlayer"))
			Expect(hardware.Domain).To(Equal("testing.com"))
			Expect(hardware.BareMetalInstanceFlag).To(Equal(1))
			Expect(hardware.GlobalIdentifier).To(Equal("abcdefg"))
		})
	})

	Context("#GetObject", func() {
		BeforeEach(func() {
			fakeClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Hardware_Service_createObject.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("sucessfully retrieves SoftLayer_Virtual_Guest instance", func() {
			hardware, err := hardwareService.GetObject("abcdefg")
			Expect(err).ToNot(HaveOccurred())
			Expect(hardware.Id).To(Equal(123))
			Expect(hardware.Hostname).To(Equal("softlayer"))
			Expect(hardware.Domain).To(Equal("testing.com"))
			Expect(hardware.BareMetalInstanceFlag).To(Equal(1))
			Expect(hardware.GlobalIdentifier).To(Equal("abcdefg"))
		})
	})
})
