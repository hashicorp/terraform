package services_test

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	slclientfakes "github.com/TheWeatherCompany/softlayer-go/client/fakes"
	softlayer "github.com/TheWeatherCompany/softlayer-go/softlayer"
	testhelpers "github.com/TheWeatherCompany/softlayer-go/test_helpers"
)

var _ = Describe("SoftLayer_Product_Package", func() {
	var (
		username, apiKey string

		fakeClient *slclientfakes.FakeSoftLayerClient

		productPackageService softlayer.SoftLayer_Product_Package_Service
		err                   error
	)

	BeforeEach(func() {
		username = os.Getenv("SL_USERNAME")
		Expect(username).ToNot(Equal(""))

		apiKey = os.Getenv("SL_API_KEY")
		Expect(apiKey).ToNot(Equal(""))

		fakeClient = slclientfakes.NewFakeSoftLayerClient(username, apiKey)
		Expect(fakeClient).ToNot(BeNil())

		productPackageService, err = fakeClient.GetSoftLayer_Product_Package_Service()
		Expect(err).ToNot(HaveOccurred())
		Expect(productPackageService).ToNot(BeNil())
	})

	Context("#GetName", func() {
		It("returns the name for the service", func() {
			name := productPackageService.GetName()
			Expect(name).To(Equal("SoftLayer_Product_Package"))
		})
	})

	Context("#GetItemPrices", func() {
		BeforeEach(func() {
			fakeClient.FakeHttpClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Product_Package_getItemPrices.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns an array of datatypes.SoftLayer_Product_Item_Price", func() {
			itemPrices, err := productPackageService.GetItemPrices(0)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(itemPrices)).To(Equal(1))
			Expect(itemPrices[0].Id).To(Equal(123))
			Expect(itemPrices[0].Item.Id).To(Equal(456))
		})

		Context("when HTTP client returns error codes 40x or 50x", func() {
			It("fails for error code 40x", func() {
				errorCodes := []int{400, 401, 499}
				for _, errorCode := range errorCodes {
					fakeClient.FakeHttpClient.DoRawHttpRequestInt = errorCode

					_, err := productPackageService.GetItemPrices(0)
					Expect(err).To(HaveOccurred())
				}
			})

			It("fails for error code 50x", func() {
				errorCodes := []int{500, 501, 599}
				for _, errorCode := range errorCodes {
					fakeClient.FakeHttpClient.DoRawHttpRequestInt = errorCode

					_, err := productPackageService.GetItemPrices(0)
					Expect(err).To(HaveOccurred())
				}
			})
		})
	})

	Context("#GetItemPricesBySize", func() {
		BeforeEach(func() {
			fakeClient.FakeHttpClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Product_Package_getItemPrices.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns an array of datatypes.SoftLayer_Product_Item_Price", func() {
			itemPrices, err := productPackageService.GetItemPricesBySize(222, 20)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(itemPrices)).To(Equal(1))
			Expect(itemPrices[0].Id).To(Equal(123))
			Expect(itemPrices[0].Item.Id).To(Equal(456))
		})

		Context("when HTTP client returns error codes 40x or 50x", func() {
			It("fails for error code 40x", func() {
				errorCodes := []int{400, 401, 499}
				for _, errorCode := range errorCodes {
					fakeClient.FakeHttpClient.DoRawHttpRequestInt = errorCode

					_, err := productPackageService.GetItemPricesBySize(222, 20)
					Expect(err).To(HaveOccurred())
				}
			})

			It("fails for error code 50x", func() {
				errorCodes := []int{500, 501, 599}
				for _, errorCode := range errorCodes {
					fakeClient.FakeHttpClient.DoRawHttpRequestInt = errorCode

					_, err := productPackageService.GetItemPricesBySize(222, 20)
					Expect(err).To(HaveOccurred())
				}
			})
		})
	})

	Context("#GetItems", func() {
		BeforeEach(func() {
			fakeClient.FakeHttpClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Product_Package_getItems.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns an array of datatypes.SoftLayer_Product_Item", func() {
			productItems, err := productPackageService.GetItems(222)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(productItems)).To(Equal(2))
			Expect(productItems[0].Id).To(Equal(123))
			Expect(productItems[0].Prices[0].Id).To(Equal(456))
		})

		Context("when HTTP client returns error codes 40x or 50x", func() {
			It("fails for error code 40x", func() {
				errorCodes := []int{400, 401, 499}
				for _, errorCode := range errorCodes {
					fakeClient.FakeHttpClient.DoRawHttpRequestInt = errorCode

					_, err := productPackageService.GetItems(222)
					Expect(err).To(HaveOccurred())
				}
			})

			It("fails for error code 50x", func() {
				errorCodes := []int{500, 501, 599}
				for _, errorCode := range errorCodes {
					fakeClient.FakeHttpClient.DoRawHttpRequestInt = errorCode

					_, err := productPackageService.GetItems(222)
					Expect(err).To(HaveOccurred())
				}
			})
		})
	})

	Context("#GetItemsByType", func() {
		BeforeEach(func() {
			fakeClient.FakeHttpClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Product_Package_getAllObjects_virtual_server.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns an array of datatypes.SoftLayer_Product_Item", func() {
			fakeClient.FakeHttpClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Product_Package_getItems.json")
			Expect(err).ToNot(HaveOccurred())

			productItems, err := productPackageService.GetItemsByType("VIRTUAL_SERVER_INSTANCE")
			Expect(err).ToNot(HaveOccurred())
			Expect(len(productItems)).To(Equal(2))
			Expect(productItems[0].Id).To(Equal(123))
			Expect(productItems[0].Prices[0].Id).To(Equal(456))
		})

		Context("when HTTP client returns error codes 40x or 50x", func() {
			It("fails for error code 40x", func() {
				errorCodes := []int{400, 401, 499}
				for _, errorCode := range errorCodes {
					fakeClient.FakeHttpClient.DoRawHttpRequestInt = errorCode

					_, err := productPackageService.GetItemsByType("VIRTUAL_SERVER_INSTANCE")
					Expect(err).To(HaveOccurred())
				}
			})

			It("fails for error code 50x", func() {
				errorCodes := []int{500, 501, 599}
				for _, errorCode := range errorCodes {
					fakeClient.FakeHttpClient.DoRawHttpRequestInt = errorCode

					_, err := productPackageService.GetItemsByType("VIRTUAL_SERVER_INSTANCE")
					Expect(err).To(HaveOccurred())
				}
			})
		})
	})

	Context("#GetPackagesByType", func() {
		BeforeEach(func() {
			fakeClient.FakeHttpClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Product_Package_getAllObjects_virtual_server.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns an array of datatypes.Softlayer_Product_Package", func() {
			productPackages, err := productPackageService.GetPackagesByType("VIRTUAL_SERVER_INSTANCE")
			Expect(err).ToNot(HaveOccurred())
			Expect(len(productPackages)).To(Equal(4))
			Expect(productPackages[0].Id).To(Equal(200))
			Expect(productPackages[0].Name).To(Equal("Cloud Server 1"))
		})

		It("skips packaged marked OUTLET", func() {
			fakeClient.FakeHttpClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Product_Package_getAllObjects_virtual_server_with_OUTLET.json")

			productPackages, err := productPackageService.GetPackagesByType("VIRTUAL_SERVER_INSTANCE")
			Expect(err).ToNot(HaveOccurred())
			Expect(len(productPackages)).To(Equal(3)) // OUTLET should be skipped
			Expect(productPackages[0].Id).To(Equal(202))
			Expect(productPackages[0].Name).To(Equal("Cloud Server 2"))
		})

		Context("when HTTP client returns error codes 40x or 50x", func() {
			It("fails for error code 40x", func() {
				errorCodes := []int{400, 401, 499}
				for _, errorCode := range errorCodes {
					fakeClient.FakeHttpClient.DoRawHttpRequestInt = errorCode

					_, err := productPackageService.GetPackagesByType("VIRTUAL_SERVER_INSTANCE")
					Expect(err).To(HaveOccurred())
				}
			})

			It("fails for error code 50x", func() {
				errorCodes := []int{500, 501, 599}
				for _, errorCode := range errorCodes {
					fakeClient.FakeHttpClient.DoRawHttpRequestInt = errorCode

					_, err := productPackageService.GetPackagesByType("VIRTUAL_SERVER_INSTANCE")
					Expect(err).To(HaveOccurred())
				}
			})
		})
	})

	Context("#GetOnePackageByType", func() {
		BeforeEach(func() {
			fakeClient.FakeHttpClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Product_Package_getAllObjects_virtual_server.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("reports error when NO product packages are found", func() {
			fakeClient.FakeHttpClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Product_Package_getAllObjects_virtual_server_empty.json")

			GinkgoWriter.Write(fakeClient.FakeHttpClient.DoRawHttpRequestResponse)

			_, err := productPackageService.GetOnePackageByType("SOME_TYPE")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("No packages available for type 'SOME_TYPE'."))
		})

		It("returns datatypes.Softlayer_Product_Package", func() {
			fakeClient.FakeHttpClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Product_Package_getAllObjects_virtual_server.json")

			productPackage, err := productPackageService.GetOnePackageByType("VIRTUAL_SERVER_INSTANCE")
			Expect(err).ToNot(HaveOccurred())
			Expect(productPackage.Id).To(Equal(200))
			Expect(productPackage.Name).To(Equal("Cloud Server 1"))
		})

		Context("when HTTP client returns error codes 40x or 50x", func() {
			It("fails for error code 40x", func() {
				errorCodes := []int{400, 401, 499}
				for _, errorCode := range errorCodes {
					fakeClient.FakeHttpClient.DoRawHttpRequestInt = errorCode

					_, err := productPackageService.GetOnePackageByType("VIRTUAL_SERVER_INSTANCE")
					Expect(err).To(HaveOccurred())
				}
			})

			It("fails for error code 50x", func() {
				errorCodes := []int{500, 501, 599}
				for _, errorCode := range errorCodes {
					fakeClient.FakeHttpClient.DoRawHttpRequestInt = errorCode

					_, err := productPackageService.GetOnePackageByType("VIRTUAL_SERVER_INSTANCE")
					Expect(err).To(HaveOccurred())
				}
			})
		})
	})
})
