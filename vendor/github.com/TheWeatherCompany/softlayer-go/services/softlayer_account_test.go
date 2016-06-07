package services_test

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	slclientfakes "github.com/TheWeatherCompany/softlayer-go/client/fakes"
	softlayer "github.com/TheWeatherCompany/softlayer-go/softlayer"
	testhelpers "github.com/TheWeatherCompany/softlayer-go/test_helpers"
)

var _ = Describe("SoftLayer_Account_Service", func() {
	var (
		username, apiKey string

		fakeClient *slclientfakes.FakeSoftLayerClient

		accountService softlayer.SoftLayer_Account_Service
		err            error
	)

	BeforeEach(func() {
		username = os.Getenv("SL_USERNAME")
		Expect(username).ToNot(Equal(""))

		apiKey = os.Getenv("SL_API_KEY")
		Expect(apiKey).ToNot(Equal(""))

		fakeClient = slclientfakes.NewFakeSoftLayerClient(username, apiKey)
		Expect(fakeClient).ToNot(BeNil())

		accountService, err = fakeClient.GetSoftLayer_Account_Service()
		Expect(err).ToNot(HaveOccurred())
		Expect(accountService).ToNot(BeNil())
	})

	Context("#GetName", func() {
		It("returns the name for the service", func() {
			name := accountService.GetName()
			Expect(name).To(Equal("SoftLayer_Account"))
		})
	})

	Context("#GetAccountStatus", func() {
		BeforeEach(func() {
			fakeClient.FakeHttpClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Account_Service_getAccountStatus.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns an instance of datatypes.SoftLayer_Account_Status that is Active", func() {
			accountStatus, err := accountService.GetAccountStatus()
			Expect(err).ToNot(HaveOccurred())
			Expect(accountStatus.Id).ToNot(Equal(0))
			Expect(accountStatus.Name).To(Equal("Active"))
		})

		Context("when HTTP client returns error codes 40x or 50x", func() {
			It("fails for error code 40x", func() {
				errorCodes := []int{400, 401, 499}
				for _, errorCode := range errorCodes {
					fakeClient.FakeHttpClient.DoRawHttpRequestInt = errorCode

					_, err := accountService.GetAccountStatus()
					Expect(err).To(HaveOccurred())
				}
			})

			It("fails for error code 50x", func() {
				errorCodes := []int{500, 501, 599}
				for _, errorCode := range errorCodes {
					fakeClient.FakeHttpClient.DoRawHttpRequestInt = errorCode

					_, err := accountService.GetAccountStatus()
					Expect(err).To(HaveOccurred())
				}
			})
		})
	})

	Context("#GetVirtualGuests", func() {
		BeforeEach(func() {
			fakeClient.FakeHttpClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Account_Service_getVirtualGuests.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns an array of datatypes.SoftLayer_Virtual_Guest", func() {
			virtualGuests, err := accountService.GetVirtualGuests()
			Expect(err).ToNot(HaveOccurred())
			Expect(virtualGuests).ToNot(BeNil())
		})

		Context("when HTTP client returns error codes 40x or 50x", func() {
			It("fails for error code 40x", func() {
				errorCodes := []int{400, 401, 499}
				for _, errorCode := range errorCodes {
					fakeClient.FakeHttpClient.DoRawHttpRequestInt = errorCode

					_, err := accountService.GetVirtualGuests()
					Expect(err).To(HaveOccurred())
				}
			})

			It("fails for error code 50x", func() {
				errorCodes := []int{500, 501, 599}
				for _, errorCode := range errorCodes {
					fakeClient.FakeHttpClient.DoRawHttpRequestInt = errorCode

					_, err := accountService.GetVirtualGuests()
					Expect(err).To(HaveOccurred())
				}
			})
		})
	})

	Context("#GetVirtualGuestsByFilter", func() {
		BeforeEach(func() {
			fakeClient.FakeHttpClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Account_Service_getVirtualGuests.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns an array of datatypes.SoftLayer_Virtual_Guest", func() {
			virtualGuests, err := accountService.GetVirtualGuestsByFilter("fake-filter")
			Expect(err).ToNot(HaveOccurred())
			Expect(virtualGuests).ToNot(BeNil())
		})

		Context("when HTTP client returns error codes 40x or 50x", func() {
			It("fails for error code 40x", func() {
				errorCodes := []int{400, 401, 499}
				for _, errorCode := range errorCodes {
					fakeClient.FakeHttpClient.DoRawHttpRequestInt = errorCode

					_, err := accountService.GetVirtualGuests()
					Expect(err).To(HaveOccurred())
				}
			})

			It("fails for error code 50x", func() {
				errorCodes := []int{500, 501, 599}
				for _, errorCode := range errorCodes {
					fakeClient.FakeHttpClient.DoRawHttpRequestInt = errorCode

					_, err := accountService.GetVirtualGuests()
					Expect(err).To(HaveOccurred())
				}
			})
		})
	})

	Context("#GetNetworkStorage", func() {
		BeforeEach(func() {
			fakeClient.FakeHttpClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Account_Service_getNetworkStorage.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns an array of datatypes.SoftLayer_Network_Storage", func() {
			networkStorage, err := accountService.GetNetworkStorage()
			Expect(err).ToNot(HaveOccurred())
			Expect(networkStorage).ToNot(BeNil())
		})

		Context("when HTTP client returns error codes 40x or 50x", func() {
			It("fails for error code 40x", func() {
				errorCodes := []int{400, 401, 499}
				for _, errorCode := range errorCodes {
					fakeClient.FakeHttpClient.DoRawHttpRequestInt = errorCode

					_, err := accountService.GetNetworkStorage()
					Expect(err).To(HaveOccurred())
				}
			})

			It("fails for error code 50x", func() {
				errorCodes := []int{500, 501, 599}
				for _, errorCode := range errorCodes {
					fakeClient.FakeHttpClient.DoRawHttpRequestInt = errorCode

					_, err := accountService.GetNetworkStorage()
					Expect(err).To(HaveOccurred())
				}
			})
		})
	})

	Context("#GetIscsiNetworkStorage", func() {
		BeforeEach(func() {
			fakeClient.FakeHttpClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Account_Service_getNetworkStorage.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns an array of datatypes.SoftLayer_Network_Storage", func() {
			iscsiNetworkStorage, err := accountService.GetIscsiNetworkStorage()
			Expect(err).ToNot(HaveOccurred())
			Expect(iscsiNetworkStorage).ToNot(BeNil())
		})

		Context("when HTTP client returns error codes 40x or 50x", func() {
			It("fails for error code 40x", func() {
				errorCodes := []int{400, 401, 499}
				for _, errorCode := range errorCodes {
					fakeClient.FakeHttpClient.DoRawHttpRequestInt = errorCode

					_, err := accountService.GetIscsiNetworkStorage()
					Expect(err).To(HaveOccurred())
				}
			})

			It("fails for error code 50x", func() {
				errorCodes := []int{500, 501, 599}
				for _, errorCode := range errorCodes {
					fakeClient.FakeHttpClient.DoRawHttpRequestInt = errorCode

					_, err := accountService.GetIscsiNetworkStorage()
					Expect(err).To(HaveOccurred())
				}
			})
		})
	})

	Context("#GetIscsiNetworkStorageWithFilter", func() {
		BeforeEach(func() {
			fakeClient.FakeHttpClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Account_Service_getNetworkStorage.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns an array of datatypes.SoftLayer_Network_Storage", func() {
			iscsiNetworkStorage, err := accountService.GetIscsiNetworkStorageWithFilter("fake-filter")
			Expect(err).ToNot(HaveOccurred())
			Expect(iscsiNetworkStorage).ToNot(BeNil())
		})

		Context("when HTTP client returns error codes 40x or 50x", func() {
			It("fails for error code 40x", func() {
				errorCodes := []int{400, 401, 499}
				for _, errorCode := range errorCodes {
					fakeClient.FakeHttpClient.DoRawHttpRequestInt = errorCode

					_, err := accountService.GetIscsiNetworkStorageWithFilter("fake-filter")
					Expect(err).To(HaveOccurred())
				}
			})

			It("fails for error code 50x", func() {
				errorCodes := []int{500, 501, 599}
				for _, errorCode := range errorCodes {
					fakeClient.FakeHttpClient.DoRawHttpRequestInt = errorCode

					_, err := accountService.GetIscsiNetworkStorageWithFilter("fake-filter")
					Expect(err).To(HaveOccurred())
				}
			})
		})
	})

	Context("#GetVirtualDiskImages", func() {
		BeforeEach(func() {
			fakeClient.FakeHttpClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Account_Service_getVirtualDiskImages.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns an array of datatypes.SoftLayer_Virtual_Disk_Image", func() {
			virtualDiskImages, err := accountService.GetVirtualDiskImages()
			Expect(err).ToNot(HaveOccurred())
			Expect(virtualDiskImages).ToNot(BeNil())
		})

		Context("when HTTP client returns error codes 40x or 50x", func() {
			It("fails for error code 40x", func() {
				errorCodes := []int{400, 401, 499}
				for _, errorCode := range errorCodes {
					fakeClient.FakeHttpClient.DoRawHttpRequestInt = errorCode

					_, err := accountService.GetVirtualDiskImages()
					Expect(err).To(HaveOccurred())
				}
			})

			It("fails for error code 50x", func() {
				errorCodes := []int{500, 501, 599}
				for _, errorCode := range errorCodes {
					fakeClient.FakeHttpClient.DoRawHttpRequestInt = errorCode

					_, err := accountService.GetVirtualDiskImages()
					Expect(err).To(HaveOccurred())
				}
			})
		})
	})

	Context("#GetVirtualDiskImagesWithFilter", func() {
		BeforeEach(func() {
			fakeClient.FakeHttpClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Account_Service_getVirtualDiskImagesWithFilter.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns an array of datatypes.SoftLayer_Virtual_Disk_Image", func() {
			virtualDiskImages, err := accountService.GetVirtualDiskImagesWithFilter(`{"correct-filter":"whatever"}`)
			Expect(err).ToNot(HaveOccurred())
			Expect(virtualDiskImages).ToNot(BeNil())
		})

		It("returns an error due to failed Json validation", func() {
			_, err := accountService.GetVirtualDiskImagesWithFilter(`{{"wrong-filter":"whatever"}`)
			Expect(err).To(HaveOccurred())
		})

		Context("when HTTP client returns error codes 40x or 50x", func() {
			It("fails for error code 40x", func() {
				errorCodes := []int{400, 401, 499}
				for _, errorCode := range errorCodes {
					fakeClient.FakeHttpClient.DoRawHttpRequestInt = errorCode

					_, err := accountService.GetVirtualDiskImagesWithFilter(`{{"wrong-filter":"whatever"}`)
					Expect(err).To(HaveOccurred())
				}
			})

			It("fails for error code 50x", func() {
				errorCodes := []int{500, 501, 599}
				for _, errorCode := range errorCodes {
					fakeClient.FakeHttpClient.DoRawHttpRequestInt = errorCode

					_, err := accountService.GetVirtualDiskImagesWithFilter(`{{"wrong-filter":"whatever"}`)
					Expect(err).To(HaveOccurred())
				}
			})
		})
	})

	Context("#GetSshKeys", func() {
		BeforeEach(func() {
			fakeClient.FakeHttpClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Account_Service_getSshKeys.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns an array of datatypes.SoftLayer_Ssh_Key", func() {
			sshKeys, err := accountService.GetSshKeys()
			Expect(err).ToNot(HaveOccurred())
			Expect(sshKeys).ToNot(BeNil())
		})

		Context("when HTTP client returns error codes 40x or 50x", func() {
			It("fails for error code 40x", func() {
				errorCodes := []int{400, 401, 499}
				for _, errorCode := range errorCodes {
					fakeClient.FakeHttpClient.DoRawHttpRequestInt = errorCode

					_, err := accountService.GetSshKeys()
					Expect(err).To(HaveOccurred())
				}
			})

			It("fails for error code 50x", func() {
				errorCodes := []int{500, 501, 599}
				for _, errorCode := range errorCodes {
					fakeClient.FakeHttpClient.DoRawHttpRequestInt = errorCode

					_, err := accountService.GetSshKeys()
					Expect(err).To(HaveOccurred())
				}
			})
		})

	})

	Context("#GetBlockDeviceTemplateGroups", func() {
		BeforeEach(func() {
			fakeClient.FakeHttpClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Account_Service_getBlockDeviceTemplateGroups.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns an array of datatypes.SoftLayer_Virtual_Guest_Block_Device_Template_Group", func() {
			groups, err := accountService.GetBlockDeviceTemplateGroups()
			Expect(err).ToNot(HaveOccurred())
			Expect(groups).ToNot(BeNil())
		})

		Context("when HTTP client returns error codes 40x or 50x", func() {
			It("fails for error code 40x", func() {
				errorCodes := []int{400, 401, 499}
				for _, errorCode := range errorCodes {
					fakeClient.FakeHttpClient.DoRawHttpRequestInt = errorCode

					_, err := accountService.GetBlockDeviceTemplateGroups()
					Expect(err).To(HaveOccurred())
				}
			})

			It("fails for error code 50x", func() {
				errorCodes := []int{500, 501, 599}
				for _, errorCode := range errorCodes {
					fakeClient.FakeHttpClient.DoRawHttpRequestInt = errorCode

					_, err := accountService.GetBlockDeviceTemplateGroups()
					Expect(err).To(HaveOccurred())
				}
			})
		})
	})

	Context("#GetBlockDeviceTemplateGroupsWithFilter", func() {
		BeforeEach(func() {
			fakeClient.FakeHttpClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Account_Service_getBlockDeviceTemplateGroups.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns an array of datatypes.SoftLayer_Virtual_Guest_Block_Device_Template_Group", func() {
			groups, err := accountService.GetBlockDeviceTemplateGroupsWithFilter(`{"correct-filter":"whatever"}`)
			Expect(err).ToNot(HaveOccurred())
			Expect(groups).ToNot(BeNil())
		})

		It("returns an error due to failed Json validation", func() {
			_, err := accountService.GetBlockDeviceTemplateGroupsWithFilter(`{{"wrong-filter":"whatever"}`)
			Expect(err).To(HaveOccurred())
		})

		Context("when HTTP client returns error codes 40x or 50x", func() {
			It("fails for error code 40x", func() {
				errorCodes := []int{400, 401, 499}
				for _, errorCode := range errorCodes {
					fakeClient.FakeHttpClient.DoRawHttpRequestInt = errorCode

					_, err := accountService.GetBlockDeviceTemplateGroupsWithFilter(`{"correct-filter":"whatever"}`)
					Expect(err).To(HaveOccurred())
				}
			})

			It("fails for error code 50x", func() {
				errorCodes := []int{500, 501, 599}
				for _, errorCode := range errorCodes {
					fakeClient.FakeHttpClient.DoRawHttpRequestInt = errorCode

					_, err := accountService.GetBlockDeviceTemplateGroupsWithFilter(`{"correct-filter":"whatever"}`)
					Expect(err).To(HaveOccurred())
				}
			})
		})
	})

	Context("#GetDatacentersWithSubnetAllocations", func() {
		BeforeEach(func() {
			fakeClient.FakeHttpClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Account_Service_getDatacentersWithSubnetAllocations.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns an array of datatypes.SoftLayer_Virtual_Location", func() {
			locations, err := accountService.GetBlockDeviceTemplateGroups()
			Expect(err).ToNot(HaveOccurred())
			Expect(locations).ToNot(BeNil())
			Expect(len(locations)).To(BeNumerically(">", 0))
		})

		Context("when HTTP client returns error codes 40x or 50x", func() {
			It("fails for error code 40x", func() {
				errorCodes := []int{400, 401, 499}
				for _, errorCode := range errorCodes {
					fakeClient.FakeHttpClient.DoRawHttpRequestInt = errorCode

					_, err := accountService.GetBlockDeviceTemplateGroups()
					Expect(err).To(HaveOccurred())
				}
			})

			It("fails for error code 50x", func() {
				errorCodes := []int{500, 501, 599}
				for _, errorCode := range errorCodes {
					fakeClient.FakeHttpClient.DoRawHttpRequestInt = errorCode

					_, err := accountService.GetBlockDeviceTemplateGroups()
					Expect(err).To(HaveOccurred())
				}
			})
		})
	})

	Context("#GetHardware", func() {
		BeforeEach(func() {
			fakeClient.FakeHttpClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Account_Service_getHardware.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns an array of datatypes.SoftLayer_Hardware", func() {
			hardwares, err := accountService.GetHardware()
			Expect(err).ToNot(HaveOccurred())
			Expect(hardwares).ToNot(BeNil())
			Expect(len(hardwares)).To(BeNumerically(">", 0))
		})

		Context("when HTTP client returns error codes 40x or 50x", func() {
			It("fails for error code 40x", func() {
				errorCodes := []int{400, 401, 499}
				for _, errorCode := range errorCodes {
					fakeClient.FakeHttpClient.DoRawHttpRequestInt = errorCode

					_, err := accountService.GetHardware()
					Expect(err).To(HaveOccurred())
				}
			})

			It("fails for error code 50x", func() {
				errorCodes := []int{500, 501, 599}
				for _, errorCode := range errorCodes {
					fakeClient.FakeHttpClient.DoRawHttpRequestInt = errorCode

					_, err := accountService.GetHardware()
					Expect(err).To(HaveOccurred())
				}
			})
		})
	})

	Context("#GetApplicationDeliveryControllersWithFilter", func() {
		BeforeEach(func() {
			fakeClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Account_Service_getApplicationDeliveryControllers.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns an array of datatypes.SoftLayer_Network_Application_Delivery_Controller", func() {
			ObjectFilter := string(`{"iscsiNetworkStorage":{"billingItem":{"orderItem":{"order":{"id":{"operation":123}}}}}}`)
			applicationDeliveryControllers, err := accountService.GetApplicationDeliveryControllersWithFilter(ObjectFilter)
			Expect(err).ToNot(HaveOccurred())
			Expect(applicationDeliveryControllers).ToNot(BeNil())
		})
	})
})
