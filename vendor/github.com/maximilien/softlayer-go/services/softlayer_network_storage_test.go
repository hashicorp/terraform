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

var _ = Describe("SoftLayer_Network_Storage", func() {
	var (
		username, apiKey string

		fakeClient *slclientfakes.FakeSoftLayerClient

		volume                datatypes.SoftLayer_Network_Storage
		networkStorageService softlayer.SoftLayer_Network_Storage_Service
		err                   error
	)

	BeforeEach(func() {
		username = os.Getenv("SL_USERNAME")
		Expect(username).ToNot(Equal(""))

		apiKey = os.Getenv("SL_API_KEY")
		Expect(apiKey).ToNot(Equal(""))

		fakeClient = slclientfakes.NewFakeSoftLayerClient(username, apiKey)
		Expect(fakeClient).ToNot(BeNil())

		networkStorageService, err = fakeClient.GetSoftLayer_Network_Storage_Service()
		Expect(err).ToNot(HaveOccurred())
		Expect(networkStorageService).ToNot(BeNil())

		volume = datatypes.SoftLayer_Network_Storage{}
	})

	Context("#GetName", func() {
		It("returns the name for the service", func() {
			name := networkStorageService.GetName()
			Expect(name).To(Equal("SoftLayer_Network_Storage"))
		})
	})

	Context("#CreateIscsiVolume", func() {
		BeforeEach(func() {
			fakeClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Network_Storage_Service_getIscsiVolume.json")
			Expect(err).ToNot(HaveOccurred())
		})
		It("fails with error if the volume size is negative", func() {
			volume, err = networkStorageService.CreateIscsiVolume(-1, "fake-location")
			Expect(err).To(HaveOccurred())
		})
	})

	Context("#GetIscsiVolume", func() {
		BeforeEach(func() {
			fakeClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Network_Storage_Service_getIscsiVolume.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns the iSCSI volume object based on volume id", func() {
			volume, err = networkStorageService.GetIscsiVolume(1)
			Expect(err).ToNot(HaveOccurred())
			Expect(volume.Id).To(Equal(1))
			Expect(volume.Username).To(Equal("test_username"))
			Expect(volume.Password).To(Equal("test_password"))
			Expect(volume.CapacityGb).To(Equal(20))
			Expect(volume.ServiceResourceBackendIpAddress).To(Equal("1.1.1.1"))
		})
	})

	Context("#HasAllowedVirtualGuest", func() {
		It("virtual guest allows to access volume", func() {
			fakeClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Network_Storage_Service_getAllowedVirtualGuests.json")
			Expect(err).ToNot(HaveOccurred())
			_, err := networkStorageService.HasAllowedVirtualGuest(123, 456)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("#AttachIscsiVolume", func() {
		It("Allow access to storage from virutal guest", func() {
			virtualGuest := datatypes.SoftLayer_Virtual_Guest{
				AccountId:                    123456,
				DedicatedAccountHostOnlyFlag: false,
				Domain: "softlayer.com",
				FullyQualifiedDomainName: "fake.softlayer.com",
				Hostname:                 "fake-hostname",
				Id:                       1234567,
				MaxCpu:                   2,
				MaxCpuUnits:              "CORE",
				MaxMemory:                1024,
				StartCpus:                2,
				StatusId:                 1001,
				Uuid:                     "fake-uuid",
				GlobalIdentifier:         "fake-globalIdentifier",
				PrimaryBackendIpAddress:  "fake-primary-backend-ip",
				PrimaryIpAddress:         "fake-primary-ip",
			}
			fakeClient.DoRawHttpRequestResponse = []byte("true")
			resp, err := networkStorageService.AttachIscsiVolume(virtualGuest, 123)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp).To(Equal(true))
		})
	})

	Context("#DetachIscsiVolume", func() {
		It("Revoke access to storage from virtual guest", func() {
			virtualGuest := datatypes.SoftLayer_Virtual_Guest{
				AccountId:                    123456,
				DedicatedAccountHostOnlyFlag: false,
				Domain: "softlayer.com",
				FullyQualifiedDomainName: "fake.softlayer.com",
				Hostname:                 "fake-hostname",
				Id:                       1234567,
				MaxCpu:                   2,
				MaxCpuUnits:              "CORE",
				MaxMemory:                1024,
				StartCpus:                2,
				StatusId:                 1001,
				Uuid:                     "fake-uuid",
				GlobalIdentifier:         "fake-globalIdentifier",
				PrimaryBackendIpAddress:  "fake-primary-backend-ip",
				PrimaryIpAddress:         "fake-primary-ip",
			}
			volume.Id = 1234567
			fakeClient.DoRawHttpRequestResponse = []byte("true")
			err = networkStorageService.DetachIscsiVolume(virtualGuest, volume.Id)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("#DeleteObject", func() {
		BeforeEach(func() {
			volume.Id = 1234567
		})

		It("sucessfully deletes the SoftLayer_Network_Storage volume", func() {
			fakeClient.DoRawHttpRequestResponse = []byte("true")
			deleted, err := networkStorageService.DeleteObject(volume.Id)
			Expect(err).ToNot(HaveOccurred())
			Expect(deleted).To(BeTrue())
		})

		It("fails to delete the SoftLayer_Network_Storage volume", func() {
			fakeClient.DoRawHttpRequestResponse = []byte("false")
			deleted, err := networkStorageService.DeleteObject(volume.Id)
			Expect(err).To(HaveOccurred())
			Expect(deleted).To(BeFalse())
		})
	})

})
