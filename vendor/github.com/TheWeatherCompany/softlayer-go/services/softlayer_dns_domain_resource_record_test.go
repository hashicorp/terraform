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

var _ = Describe("SoftLayer_Dns_Domain_Record", func() {
	var (
		username, apiKey string

		fakeClient *slclientfakes.FakeSoftLayerClient

		dnsDomainResourceRecordService softlayer.SoftLayer_Dns_Domain_ResourceRecord_Service
		err                            error
	)

	BeforeEach(func() {
		username = os.Getenv("SL_USERNAME")
		Expect(username).ToNot(Equal(""))

		apiKey = os.Getenv("SL_API_KEY")
		Expect(apiKey).ToNot(Equal(""))

		fakeClient = slclientfakes.NewFakeSoftLayerClient(username, apiKey)
		Expect(fakeClient).ToNot(BeNil())

		dnsDomainResourceRecordService, err = fakeClient.GetSoftLayer_Dns_Domain_ResourceRecord_Service()
		Expect(err).ToNot(HaveOccurred())
		Expect(dnsDomainResourceRecordService).ToNot(BeNil())
	})

	Context("#GetName", func() {
		It("returns the name for the service", func() {
			name := dnsDomainResourceRecordService.GetName()
			Expect(name).To(Equal("SoftLayer_Dns_Domain_ResourceRecord"))
		})
	})

	Context("#CreateObject", func() {
		var template datatypes.SoftLayer_Dns_Domain_ResourceRecord_Template

		BeforeEach(func() {
			fakeClient.FakeHttpClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Dns_Domain_ResourceRecord_Service_createObject.json")
			Expect(err).ToNot(HaveOccurred())

			template = datatypes.SoftLayer_Dns_Domain_ResourceRecord_Template{
				Data:              "testData",
				DomainId:          123,
				Expire:            99999,
				Host:              "testHost.com",
				Id:                111,
				Minimum:           1,
				MxPriority:        9,
				Refresh:           100,
				ResponsiblePerson: "someTestPerson",
				Retry:             444,
				Ttl:               222,
				Type:              "someTestType",
			}
		})

		It("creates a new SoftLayer_Dns_Domain_Record", func() {
			result, err := dnsDomainResourceRecordService.CreateObject(template)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Data).To(Equal("testData"))
			Expect(result.DomainId).To(Equal(123))
			Expect(result.Expire).To(Equal(99999))
			Expect(result.Host).To(Equal("testHost.com"))
			Expect(result.Id).To(Equal(111))
			Expect(result.Minimum).To(Equal(1))
			Expect(result.MxPriority).To(Equal(9))
			Expect(result.Refresh).To(Equal(100))
			Expect(result.ResponsiblePerson).To(Equal("someTestPerson"))
			Expect(result.Retry).To(Equal(444))
			Expect(result.Ttl).To(Equal(222))
			Expect(result.Type).To(Equal("someTestType"))
		})

		It("fails to create a resource record without mandatory parameters", func() {
			fakeClient.FakeHttpClient.DoRawHttpRequestResponse = []byte("fake")

			template := datatypes.SoftLayer_Dns_Domain_ResourceRecord_Template{
				Data: "testData",
			}

			_, err := dnsDomainResourceRecordService.CreateObject(template)
			Expect(err).To(HaveOccurred())
		})

		Context("when HTTP client returns error codes 40x or 50x", func() {
			It("fails for error code 40x", func() {
				errorCodes := []int{400, 401, 499}
				for _, errorCode := range errorCodes {
					fakeClient.FakeHttpClient.DoRawHttpRequestInt = errorCode

					_, err := dnsDomainResourceRecordService.CreateObject(template)
					Expect(err).To(HaveOccurred())
				}
			})

			It("fails for error code 50x", func() {
				errorCodes := []int{500, 501, 599}
				for _, errorCode := range errorCodes {
					fakeClient.FakeHttpClient.DoRawHttpRequestInt = errorCode

					_, err := dnsDomainResourceRecordService.CreateObject(template)
					Expect(err).To(HaveOccurred())
				}
			})
		})
	})

	Context("#GetObject", func() {
		BeforeEach(func() {
			fakeClient.FakeHttpClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Dns_Domain_ResourceRecord_Service_createObject.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("sucessfully retrieves SoftLayer_Dns_Domain_Record instance", func() {
			result, err := dnsDomainResourceRecordService.GetObject(111)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Data).To(Equal("testData"))
			Expect(result.DomainId).To(Equal(123))
			Expect(result.Expire).To(Equal(99999))
			Expect(result.Host).To(Equal("testHost.com"))
			Expect(result.Id).To(Equal(111))
			Expect(result.Minimum).To(Equal(1))
			Expect(result.MxPriority).To(Equal(9))
			Expect(result.Refresh).To(Equal(100))
			Expect(result.ResponsiblePerson).To(Equal("someTestPerson"))
			Expect(result.Retry).To(Equal(444))
			Expect(result.Ttl).To(Equal(222))
			Expect(result.Type).To(Equal("someTestType"))
		})

		Context("when HTTP client returns error codes 40x or 50x", func() {
			It("fails for error code 40x", func() {
				errorCodes := []int{400, 401, 499}
				for _, errorCode := range errorCodes {
					fakeClient.FakeHttpClient.DoRawHttpRequestInt = errorCode

					_, err := dnsDomainResourceRecordService.GetObject(111)
					Expect(err).To(HaveOccurred())
				}
			})

			It("fails for error code 50x", func() {
				errorCodes := []int{500, 501, 599}
				for _, errorCode := range errorCodes {
					fakeClient.FakeHttpClient.DoRawHttpRequestInt = errorCode

					_, err := dnsDomainResourceRecordService.GetObject(111)
					Expect(err).To(HaveOccurred())
				}
			})
		})
	})

	Context("#EditObject", func() {
		BeforeEach(func() {
			fakeClient.FakeHttpClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Dns_Domain_ResourceRecord_Service_editObject.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("applies changes to the existing SoftLayer_Dns_Domain_Record instance", func() {
			result, err := dnsDomainResourceRecordService.GetObject(112)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Data).To(Equal("changedData"))
			Expect(result.DomainId).To(Equal(124))
			Expect(result.Expire).To(Equal(99998))
			Expect(result.Host).To(Equal("changedHost.com"))
			Expect(result.Id).To(Equal(112))
			Expect(result.Minimum).To(Equal(2))
			Expect(result.MxPriority).To(Equal(8))
			Expect(result.Refresh).To(Equal(101))
			Expect(result.ResponsiblePerson).To(Equal("changedTestPerson"))
			Expect(result.Retry).To(Equal(445))
			Expect(result.Ttl).To(Equal(223))
			Expect(result.Type).To(Equal("changedTestType"))
		})

		Context("when HTTP client returns error codes 40x or 50x", func() {
			It("fails for error code 40x", func() {
				errorCodes := []int{400, 401, 499}
				for _, errorCode := range errorCodes {
					fakeClient.FakeHttpClient.DoRawHttpRequestInt = errorCode

					_, err := dnsDomainResourceRecordService.GetObject(112)
					Expect(err).To(HaveOccurred())
				}
			})

			It("fails for error code 50x", func() {
				errorCodes := []int{500, 501, 599}
				for _, errorCode := range errorCodes {
					fakeClient.FakeHttpClient.DoRawHttpRequestInt = errorCode

					_, err := dnsDomainResourceRecordService.GetObject(112)
					Expect(err).To(HaveOccurred())
				}
			})
		})
	})
})
