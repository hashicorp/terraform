package virtual_guest_lifecycle_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	datatypes "github.com/TheWeatherCompany/softlayer-go/data_types"
	softlayer "github.com/TheWeatherCompany/softlayer-go/softlayer"
	testhelpers "github.com/TheWeatherCompany/softlayer-go/test_helpers"
)

var _ = Describe("SoftLayer Virtual Guest Lifecycle", func() {
	var (
		err error

		accountService      softlayer.SoftLayer_Account_Service
		virtualGuestService softlayer.SoftLayer_Virtual_Guest_Service
	)

	BeforeEach(func() {
		accountService, err = testhelpers.CreateAccountService()
		Expect(err).ToNot(HaveOccurred())

		virtualGuestService, err = testhelpers.CreateVirtualGuestService()
		Expect(err).ToNot(HaveOccurred())

		testhelpers.TIMEOUT = 35 * time.Minute
		testhelpers.POLLING_INTERVAL = 10 * time.Second
	})

	XContext("SoftLayer_VirtualGuestService#setUserMetadata and SoftLayer_VirtualGuestService#configureMetadataDisk", func() {
		It("creates ssh key, VirtualGuest, waits for it to be RUNNING, set user data, configures disk, verifies user data, and delete VG", func() {
			err = testhelpers.FindAndDeleteTestSshKeys()
			Expect(err).ToNot(HaveOccurred())

			createdSshKey, publicKeyValue := testhelpers.CreateTestSshKey()
			testhelpers.WaitForCreatedSshKeyToBePresent(createdSshKey.Id)
			defer testhelpers.DeleteSshKey(createdSshKey.Id)

			virtualGuest := testhelpers.CreateVirtualGuestAndMarkItTest([]datatypes.SoftLayer_Security_Ssh_Key{createdSshKey})
			defer testhelpers.WaitForVirtualGuestToHaveNoActiveTransactionsOrToErr(virtualGuest.Id)
			defer testhelpers.CleanUpVirtualGuest(virtualGuest.Id)

			testhelpers.WaitForVirtualGuestToBeRunning(virtualGuest.Id)
			testhelpers.WaitForVirtualGuestToHaveNoActiveTransactions(virtualGuest.Id)

			startTime := time.Now()
			userMetadata := "softlayer-go test fake metadata"
			transaction := testhelpers.SetUserMetadataAndConfigureDisk(virtualGuest.Id, userMetadata)
			averageTransactionDuration, err := time.ParseDuration(transaction.TransactionStatus.AverageDuration + "m")
			Ω(err).ShouldNot(HaveOccurred())

			testhelpers.WaitForVirtualGuest(virtualGuest.Id, "RUNNING", averageTransactionDuration)
			testhelpers.WaitForVirtualGuestToHaveNoActiveTransactions(virtualGuest.Id)
			fmt.Printf("====> Set Metadata and configured disk on instance: %d in %d time\n", virtualGuest.Id, time.Since(startTime))

			testhelpers.TestUserMetadata(userMetadata, publicKeyValue)

			startTime = time.Now()
			userMetadata = "softlayer-go test MODIFIED fake metadata"
			testhelpers.SetUserMetadataAndConfigureDisk(virtualGuest.Id, userMetadata)

			testhelpers.WaitForVirtualGuest(virtualGuest.Id, "RUNNING", averageTransactionDuration)
			testhelpers.WaitForVirtualGuestToHaveNoActiveTransactions(virtualGuest.Id)
			fmt.Printf("====> Set Metadata and configured disk on instance: %d in %d time\n", virtualGuest.Id, time.Since(startTime))

			testhelpers.TestUserMetadata(userMetadata, publicKeyValue)
		})
	})
})
