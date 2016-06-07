package security_ssh_key_test

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	softlayer "github.com/TheWeatherCompany/softlayer-go/softlayer"
	testhelpers "github.com/TheWeatherCompany/softlayer-go/test_helpers"
)

var _ = Describe("SoftLayer Security SSH keys", func() {
	var (
		err             error
		securityService softlayer.SoftLayer_Security_Ssh_Key_Service
	)

	BeforeEach(func() {
		securityService, err = testhelpers.CreateSecuritySshKeyService()
		Expect(err).ToNot(HaveOccurred())

		testhelpers.TIMEOUT = 30 * time.Second
		testhelpers.POLLING_INTERVAL = 10 * time.Second
	})

	Context("SoftLayer_Security_Ssh_Key", func() {
		It("creates an SSH key, update it, and delete it", func() {
			createdSshKey, _ := testhelpers.CreateTestSshKey()

			testhelpers.WaitForCreatedSshKeyToBePresent(createdSshKey.Id)

			sshKeyService, err := testhelpers.CreateSecuritySshKeyService()
			Expect(err).ToNot(HaveOccurred())

			result, err := sshKeyService.GetObject(createdSshKey.Id)
			Expect(err).ToNot(HaveOccurred())

			Expect(result.CreateDate).ToNot(BeNil())
			Expect(result.Key).ToNot(Equal(""))
			Expect(result.Fingerprint).ToNot(Equal(""))
			Expect(result.Label).To(Equal("TEST:softlayer-go"))
			Expect(result.Notes).To(Equal("TEST:softlayer-go"))
			Expect(result.ModifyDate).To(BeNil())

			oldFingerprint := result.Fingerprint
			oldPublicKey := result.Key

			_, newPublicKey, err := testhelpers.GenerateSshKey()
			Expect(err).ToNot(HaveOccurred())

			result.Label = "TEST:softlayer-go:edited-label"
			result.Notes = "TEST:softlayer-go:edited-notes"
			result.Key = newPublicKey
			sshKeyService.EditObject(createdSshKey.Id, result)

			result2, err := sshKeyService.GetObject(createdSshKey.Id)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.CreateDate).To(Equal(result2.CreateDate))
			Expect(result2.Label).To(Equal("TEST:softlayer-go:edited-label"))
			Expect(result2.Notes).To(Equal("TEST:softlayer-go:edited-notes"))
			Expect(result2.ModifyDate).ToNot(BeNil())

			// Any attempt to change the public key will silently fail,
			// that is the behavior in SoftLayer's API.
			Expect(result2.Key).To(Equal(oldPublicKey))
			Expect(result2.Fingerprint).To(Equal(oldFingerprint))

			deleted, err := sshKeyService.DeleteObject(createdSshKey.Id)
			Expect(err).ToNot(HaveOccurred())
			Expect(deleted).To(BeTrue())

			testhelpers.WaitForDeletedSshKeyToNoLongerBePresent(createdSshKey.Id)
		})
	})
})
