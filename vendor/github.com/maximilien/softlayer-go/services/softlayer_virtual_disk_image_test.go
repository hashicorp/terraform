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

var _ = Describe("SoftLayer_Virtual_Disk_Image_Service", func() {
	var (
		username, apiKey string
		err              error

		fakeClient *slclientfakes.FakeSoftLayerClient

		virtualDiskImageService softlayer.SoftLayer_Virtual_Disk_Image_Service

		virtualDiskImage datatypes.SoftLayer_Virtual_Disk_Image
	)

	BeforeEach(func() {
		username = os.Getenv("SL_USERNAME")
		Expect(username).ToNot(Equal(""))

		apiKey = os.Getenv("SL_API_KEY")
		Expect(apiKey).ToNot(Equal(""))

		fakeClient = slclientfakes.NewFakeSoftLayerClient(username, apiKey)
		Expect(fakeClient).ToNot(BeNil())

		virtualDiskImageService, err = fakeClient.GetSoftLayer_Virtual_Disk_Image_Service()
		Expect(err).ToNot(HaveOccurred())
		Expect(virtualDiskImageService).ToNot(BeNil())

		virtualDiskImage = datatypes.SoftLayer_Virtual_Disk_Image{}
	})

	Context("#GetName", func() {
		It("returns the name for the service", func() {
			name := virtualDiskImageService.GetName()
			Expect(name).To(Equal("SoftLayer_Virtual_Disk_Image"))
		})
	})

	Context("#GetObject", func() {
		BeforeEach(func() {
			fakeClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Virtual_Disk_Image_Service_getObject.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("gets the SoftLayer_Virtual_Disk_Image instance", func() {
			virtualDiskImage, err = virtualDiskImageService.GetObject(4868344)
			Expect(err).ToNot(HaveOccurred())
			Expect(virtualDiskImage.Capacity).To(Equal(25))
			Expect(virtualDiskImage.CreateDate).ToNot(BeNil())
			Expect(virtualDiskImage.Description).To(Equal("yz-fabric-node-20140407-133340-856.softlayer.com"))
			Expect(virtualDiskImage.Id).To(Equal(4868344))
			Expect(virtualDiskImage.ModifyDate).To(BeNil())
			Expect(virtualDiskImage.Name).To(Equal("yz-fabric-node-20140407-133340-856.softlayer.com"))
			Expect(virtualDiskImage.ParentId).To(Equal(0))
			Expect(virtualDiskImage.StorageRepositoryId).To(Equal(1105002))
			Expect(virtualDiskImage.TypeId).To(Equal(241))
			Expect(virtualDiskImage.Units).To(Equal("GB"))
			Expect(virtualDiskImage.Uuid).To(Equal("8c7a8358-d9a9-4e4d-9345-6f637e10ccb7"))
		})
	})
})
