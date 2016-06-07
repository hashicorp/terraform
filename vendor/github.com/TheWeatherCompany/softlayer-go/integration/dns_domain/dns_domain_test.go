package dns_domain

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	softlayer "github.com/TheWeatherCompany/softlayer-go/softlayer"
	testhelpers "github.com/TheWeatherCompany/softlayer-go/test_helpers"
)

var _ = Describe("SoftLayer DNS domains", func() {
	var (
		err              error
		dnsDomainService softlayer.SoftLayer_Dns_Domain_Service
	)

	BeforeEach(func() {
		dnsDomainService, err = testhelpers.CreateDnsDomainService()
		Expect(err).ToNot(HaveOccurred())

		testhelpers.TIMEOUT = 30 * time.Second
		testhelpers.POLLING_INTERVAL = 10 * time.Second
	})

	Context("SoftLayer_Dns_Domain", func() {
		It("creates a DNS Domain, update it, and delete it", func() {
			createdDnsDomain := testhelpers.CreateTestDnsDomain("test.domain.name")

			testhelpers.WaitForCreatedDnsDomainToBePresent(createdDnsDomain.Id)

			result, err := dnsDomainService.GetObject(createdDnsDomain.Id)
			Expect(err).ToNot(HaveOccurred())

			Expect(result.Name).To(Equal("test.domain.name"))

			deleted, err := dnsDomainService.DeleteObject(createdDnsDomain.Id)
			Expect(err).ToNot(HaveOccurred())
			Expect(deleted).To(BeTrue())

			testhelpers.WaitForDeletedDnsDomainToNoLongerBePresent(createdDnsDomain.Id)
		})
	})
})
