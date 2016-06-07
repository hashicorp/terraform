package test_helpers

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	slclient "github.com/TheWeatherCompany/softlayer-go/client"
	datatypes "github.com/TheWeatherCompany/softlayer-go/data_types"
	softlayer "github.com/TheWeatherCompany/softlayer-go/softlayer"
)

var (
	TIMEOUT          time.Duration
	POLLING_INTERVAL time.Duration
)

const (
	TEST_NOTES_PREFIX  = "TEST:softlayer-go"
	TEST_LABEL_PREFIX  = "TEST:softlayer-go"
	DEFAULT_DATACENTER = "dal09"

	TEST_EMAIL = "testemail@sl.com"
	TEST_HOST  = "test.example.com"
	TEST_TTL   = 900

	MAX_WAIT_RETRIES = 10
	WAIT_TIME        = 5
)

func ReadJsonTestFixtures(packageName, fileName string) ([]byte, error) {
	wd, _ := os.Getwd()
	return ioutil.ReadFile(filepath.Join(wd, "..", "test_fixtures", packageName, fileName))
}

func FindTestVirtualGuests() ([]datatypes.SoftLayer_Virtual_Guest, error) {
	accountService, err := CreateAccountService()
	if err != nil {
		return []datatypes.SoftLayer_Virtual_Guest{}, err
	}

	virtualGuests, err := accountService.GetVirtualGuests()
	if err != nil {
		return []datatypes.SoftLayer_Virtual_Guest{}, err
	}

	testVirtualGuests := []datatypes.SoftLayer_Virtual_Guest{}
	for _, vGuest := range virtualGuests {
		if strings.Contains(vGuest.Notes, TEST_NOTES_PREFIX) {
			testVirtualGuests = append(testVirtualGuests, vGuest)
		}
	}

	return testVirtualGuests, nil
}

func FindTestVirtualDiskImages() ([]datatypes.SoftLayer_Virtual_Disk_Image, error) {
	accountService, err := CreateAccountService()
	if err != nil {
		return []datatypes.SoftLayer_Virtual_Disk_Image{}, err
	}

	virtualDiskImages, err := accountService.GetVirtualDiskImages()
	if err != nil {
		return []datatypes.SoftLayer_Virtual_Disk_Image{}, err
	}

	testVirtualDiskImages := []datatypes.SoftLayer_Virtual_Disk_Image{}
	for _, vDI := range virtualDiskImages {
		if strings.Contains(vDI.Description, TEST_NOTES_PREFIX) {
			testVirtualDiskImages = append(testVirtualDiskImages, vDI)
		}
	}

	return testVirtualDiskImages, nil
}

func FindTestNetworkStorage() ([]datatypes.SoftLayer_Network_Storage, error) {
	accountService, err := CreateAccountService()
	if err != nil {
		return []datatypes.SoftLayer_Network_Storage{}, err
	}

	networkStorageArray, err := accountService.GetNetworkStorage()
	if err != nil {
		return []datatypes.SoftLayer_Network_Storage{}, err
	}

	testNetworkStorageArray := []datatypes.SoftLayer_Network_Storage{}
	for _, storage := range networkStorageArray {
		if strings.Contains(storage.Notes, TEST_NOTES_PREFIX) {
			testNetworkStorageArray = append(testNetworkStorageArray, storage)
		}
	}

	return testNetworkStorageArray, nil
}

func FindTestSshKeys() ([]datatypes.SoftLayer_Security_Ssh_Key, error) {
	accountService, err := CreateAccountService()
	if err != nil {
		return []datatypes.SoftLayer_Security_Ssh_Key{}, err
	}

	sshKeys, err := accountService.GetSshKeys()
	if err != nil {
		return []datatypes.SoftLayer_Security_Ssh_Key{}, err
	}

	testSshKeys := []datatypes.SoftLayer_Security_Ssh_Key{}
	for _, key := range sshKeys {
		if key.Notes == TEST_NOTES_PREFIX {
			testSshKeys = append(testSshKeys, key)
		}
	}

	return testSshKeys, nil
}

func GetUsernameAndApiKey() (string, string, error) {
	username := os.Getenv("SL_USERNAME")
	if username == "" {
		return "", "", errors.New("SL_USERNAME environment must be set")
	}

	apiKey := os.Getenv("SL_API_KEY")
	if apiKey == "" {
		return username, "", errors.New("SL_API_KEY environment must be set")
	}

	return username, apiKey, nil
}

func GetDatacenter() string {
	datacenter := os.Getenv("SL_DATACENTER")
	if datacenter == "" {
		datacenter = DEFAULT_DATACENTER
	}
	return datacenter
}

func CreateAccountService() (softlayer.SoftLayer_Account_Service, error) {
	username, apiKey, err := GetUsernameAndApiKey()
	if err != nil {
		return nil, err
	}

	client := slclient.NewSoftLayerClient(username, apiKey)
	accountService, err := client.GetSoftLayer_Account_Service()
	if err != nil {
		return nil, err
	}

	return accountService, nil
}

func CreateVirtualGuestService() (softlayer.SoftLayer_Virtual_Guest_Service, error) {
	username, apiKey, err := GetUsernameAndApiKey()
	if err != nil {
		return nil, err
	}

	client := slclient.NewSoftLayerClient(username, apiKey)
	virtualGuestService, err := client.GetSoftLayer_Virtual_Guest_Service()
	if err != nil {
		return nil, err
	}

	return virtualGuestService, nil
}

func CreateVirtualGuestBlockDeviceTemplateGroupService() (softlayer.SoftLayer_Virtual_Guest_Block_Device_Template_Group_Service, error) {
	username, apiKey, err := GetUsernameAndApiKey()
	if err != nil {
		return nil, err
	}

	client := slclient.NewSoftLayerClient(username, apiKey)
	vgbdtgService, err := client.GetSoftLayer_Virtual_Guest_Block_Device_Template_Group_Service()
	if err != nil {
		return nil, err
	}

	return vgbdtgService, nil
}

func CreateSecuritySshKeyService() (softlayer.SoftLayer_Security_Ssh_Key_Service, error) {
	username, apiKey, err := GetUsernameAndApiKey()
	if err != nil {
		return nil, err
	}

	client := slclient.NewSoftLayerClient(username, apiKey)
	sshKeyService, err := client.GetSoftLayer_Security_Ssh_Key_Service()
	if err != nil {
		return nil, err
	}

	return sshKeyService, nil
}

func CreateProductPackageService() (softlayer.SoftLayer_Product_Package_Service, error) {
	username, apiKey, err := GetUsernameAndApiKey()
	if err != nil {
		return nil, err
	}

	client := slclient.NewSoftLayerClient(username, apiKey)
	productPackageService, err := client.GetSoftLayer_Product_Package_Service()
	if err != nil {
		return nil, err
	}

	return productPackageService, nil
}

func CreateNetworkStorageService() (softlayer.SoftLayer_Network_Storage_Service, error) {
	username, apiKey, err := GetUsernameAndApiKey()
	if err != nil {
		return nil, err
	}

	client := slclient.NewSoftLayerClient(username, apiKey)
	networkStorageService, err := client.GetSoftLayer_Network_Storage_Service()
	if err != nil {
		return nil, err
	}

	return networkStorageService, nil
}

func FindAndDeleteTestSshKeys() error {
	sshKeys, err := FindTestSshKeys()
	if err != nil {
		return err
	}

	sshKeyService, err := CreateSecuritySshKeyService()
	if err != nil {
		return err
	}

	for _, sshKey := range sshKeys {
		deleted, err := sshKeyService.DeleteObject(sshKey.Id)
		if err != nil {
			return err
		}
		if !deleted {
			return errors.New(fmt.Sprintf("Could not delete ssh key with id: %d", sshKey.Id))
		}
	}

	return nil
}

func FindAndDeleteTestVirtualGuests() ([]int, error) {
	virtualGuests, err := FindTestVirtualGuests()
	if err != nil {
		return []int{}, err
	}

	virtualGuestService, err := CreateVirtualGuestService()
	if err != nil {
		return []int{}, err
	}

	virtualGuestIds := []int{}
	for _, virtualGuest := range virtualGuests {
		virtualGuestIds = append(virtualGuestIds, virtualGuest.Id)

		deleted, err := virtualGuestService.DeleteObject(virtualGuest.Id)
		if err != nil {
			return []int{}, err
		}

		if !deleted {
			return []int{}, errors.New(fmt.Sprintf("Could not delete virtual guest with id: %d", virtualGuest.Id))
		}
	}

	return virtualGuestIds, nil
}

func MarkVirtualGuestAsTest(virtualGuest datatypes.SoftLayer_Virtual_Guest) error {
	virtualGuestService, err := CreateVirtualGuestService()
	if err != nil {
		return err
	}

	vgTemplate := datatypes.SoftLayer_Virtual_Guest{
		Notes: TEST_NOTES_PREFIX,
	}

	edited, err := virtualGuestService.EditObject(virtualGuest.Id, vgTemplate)
	if err != nil {
		return err
	}
	if edited == false {
		return errors.New(fmt.Sprintf("Could not edit virtual guest with id: %d", virtualGuest.Id))
	}

	return nil
}

func FileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	if err != nil {
		return false
	}

	return !os.IsNotExist(err)
}

func GenerateSshKey() (string, string, error) {
	return generateSshKeyUsingSshKeyGen()
}

func CreateTestSshKey() (datatypes.SoftLayer_Security_Ssh_Key, string) {
	_, testSshKeyValue, err := GenerateSshKey()
	Expect(err).ToNot(HaveOccurred())

	sshKey := datatypes.SoftLayer_Security_Ssh_Key{
		Key:   strings.Trim(string(testSshKeyValue), "\n"),
		Label: TEST_LABEL_PREFIX,
		Notes: TEST_NOTES_PREFIX,
	}

	sshKeyService, err := CreateSecuritySshKeyService()
	Expect(err).ToNot(HaveOccurred())

	fmt.Printf("----> creating ssh key in SL\n")
	createdSshKey, err := sshKeyService.CreateObject(sshKey)
	Expect(err).ToNot(HaveOccurred())

	Expect(createdSshKey.Key).To(Equal(sshKey.Key), "key")
	Expect(createdSshKey.Label).To(Equal(sshKey.Label), "label")
	Expect(createdSshKey.Notes).To(Equal(sshKey.Notes), "notes")
	Expect(createdSshKey.CreateDate).ToNot(BeNil(), "createDate")
	Expect(createdSshKey.Id).To(BeNumerically(">", 0), "id")
	Expect(createdSshKey.ModifyDate).To(BeNil(), "modifyDate")
	fmt.Printf("----> created ssh key: %d\n in SL", createdSshKey.Id)

	return createdSshKey, string(testSshKeyValue)
}

func CreateDisk(size int, location string) datatypes.SoftLayer_Network_Storage {
	networkStorageService, err := CreateNetworkStorageService()
	Expect(err).ToNot(HaveOccurred())

	fmt.Printf("----> creating new disk\n")
	disk, err := networkStorageService.CreateIscsiVolume(size, location)
	Expect(err).ToNot(HaveOccurred())
	fmt.Printf("----> created disk: %d\n", disk.Id)

	return disk
}

func CreateVirtualGuestAndMarkItTest(securitySshKeys []datatypes.SoftLayer_Security_Ssh_Key) datatypes.SoftLayer_Virtual_Guest {
	sshKeys := make([]datatypes.SshKey, len(securitySshKeys))
	for i, securitySshKey := range securitySshKeys {
		sshKeys[i] = datatypes.SshKey{Id: securitySshKey.Id}
	}

	virtualGuestTemplate := datatypes.SoftLayer_Virtual_Guest_Template{
		Hostname:  "test",
		Domain:    "softlayergo.com",
		StartCpus: 1,
		MaxMemory: 1024,
		Datacenter: datatypes.Datacenter{
			Name: GetDatacenter(),
		},
		SshKeys:                      sshKeys,
		HourlyBillingFlag:            true,
		LocalDiskFlag:                true,
		OperatingSystemReferenceCode: "UBUNTU_LATEST",
	}

	virtualGuestService, err := CreateVirtualGuestService()
	Expect(err).ToNot(HaveOccurred())

	fmt.Printf("----> creating new virtual guest\n")
	virtualGuest, err := virtualGuestService.CreateObject(virtualGuestTemplate)
	Expect(err).ToNot(HaveOccurred())
	fmt.Printf("----> created virtual guest: %d\n", virtualGuest.Id)

	WaitForVirtualGuestToBeRunning(virtualGuest.Id)
	WaitForVirtualGuestToHaveNoActiveTransactions(virtualGuest.Id)

	fmt.Printf("----> marking virtual guest with TEST:softlayer-go\n")
	err = MarkVirtualGuestAsTest(virtualGuest)
	Expect(err).ToNot(HaveOccurred(), "Could not mark virtual guest as test")
	fmt.Printf("----> marked virtual guest with TEST:softlayer-go\n")

	return virtualGuest
}

func DeleteVirtualGuest(virtualGuestId int) {
	virtualGuestService, err := CreateVirtualGuestService()
	Expect(err).ToNot(HaveOccurred())

	fmt.Printf("----> deleting virtual guest: %d\n", virtualGuestId)
	deleted, err := virtualGuestService.DeleteObject(virtualGuestId)
	Expect(err).ToNot(HaveOccurred())
	Expect(deleted).To(BeTrue(), "could not delete virtual guest")

	WaitForVirtualGuestToHaveNoActiveTransactions(virtualGuestId)
}

func CleanUpVirtualGuest(virtualGuestId int) {
	WaitForVirtualGuestToHaveNoActiveTransactions(virtualGuestId)
	DeleteVirtualGuest(virtualGuestId)
}

func DeleteSshKey(sshKeyId int) {
	sshKeyService, err := CreateSecuritySshKeyService()
	Expect(err).ToNot(HaveOccurred())

	if SshKeyPresent(sshKeyId) {
		fmt.Printf("----> deleting ssh key: %d\n", sshKeyId)
		deleted, err := sshKeyService.DeleteObject(sshKeyId)
		Expect(err).ToNot(HaveOccurred())
		Expect(deleted).To(BeTrue(), "could not delete ssh key")
	} else {
		fmt.Printf("----> ssh key %d already not present\n", sshKeyId)
	}

	WaitForDeletedSshKeyToNoLongerBePresent(sshKeyId)
}

func DeleteDisk(diskId int) {
	networkStorageService, err := CreateNetworkStorageService()
	Expect(err).ToNot(HaveOccurred())

	fmt.Printf("----> deleting disk: %d\n", diskId)
	err = networkStorageService.DeleteIscsiVolume(diskId, true)
	Expect(err).ToNot(HaveOccurred())
}

func WaitForVirtualGuest(virtualGuestId int, targetState string, timeout time.Duration) {
	virtualGuestService, err := CreateVirtualGuestService()
	Expect(err).ToNot(HaveOccurred())

	fmt.Printf("----> waiting for virtual guest: %d, until %s\n", virtualGuestId, targetState)
	Eventually(func() string {
		vgPowerState, err := virtualGuestService.GetPowerState(virtualGuestId)
		Expect(err).ToNot(HaveOccurred())
		fmt.Printf("----> virtual guest: %d, has power state: %s\n", virtualGuestId, vgPowerState.KeyName)
		return vgPowerState.KeyName
	}, timeout, POLLING_INTERVAL).Should(Equal(targetState), fmt.Sprintf("failed waiting for virtual guest to be %s", targetState))
}

func WaitForVirtualGuestToBeRunning(virtualGuestId int) {
	WaitForVirtualGuest(virtualGuestId, "RUNNING", TIMEOUT)
}

func WaitForVirtualGuestTransactionWithStatus(virtualGuestId int, status string) {
	virtualGuestService, err := CreateVirtualGuestService()
	Expect(err).ToNot(HaveOccurred())

	fmt.Printf("----> waiting for virtual guest %d to have ugrade transactions with status '%s'\n", virtualGuestId, status)
	Eventually(func() bool {
		activeTransactions, err := virtualGuestService.GetActiveTransactions(virtualGuestId)
		Expect(err).ToNot(HaveOccurred())
		for _, transaction := range activeTransactions {
			if strings.Contains(transaction.TransactionStatus.Name, status) {
				return true
			}
		}
		fmt.Printf("----> virtual guest: %d, doesn't have transactions with status '%s' yet\n", virtualGuestId, status)
		return false
	}, TIMEOUT, POLLING_INTERVAL).Should(BeTrue(), "failed waiting for virtual guest to have transactions with specifc status")
}

func WaitForVirtualGuestToHaveNoActiveTransactions(virtualGuestId int) {
	virtualGuestService, err := CreateVirtualGuestService()
	Expect(err).ToNot(HaveOccurred())

	fmt.Printf("----> waiting for virtual guest to have no active transactions pending\n")
	Eventually(func() int {
		activeTransactions, err := virtualGuestService.GetActiveTransactions(virtualGuestId)
		Expect(err).ToNot(HaveOccurred())
		fmt.Printf("----> virtual guest: %d, has %d active transactions\n", virtualGuestId, len(activeTransactions))
		return len(activeTransactions)
	}, TIMEOUT, POLLING_INTERVAL).Should(Equal(0), "failed waiting for virtual guest to have no active transactions")
}

func WaitForVirtualGuestToHaveNoActiveTransactionsOrToErr(virtualGuestId int) {
	virtualGuestService, err := CreateVirtualGuestService()
	if err != nil {
		return
	}

	fmt.Printf("----> waiting for virtual guest to have no active transactions pending\n")
	Eventually(func() int {
		activeTransactions, err := virtualGuestService.GetActiveTransactions(virtualGuestId)
		if err != nil {
			return 0
		}
		fmt.Printf("----> virtual guest: %d, has %d active transactions\n", virtualGuestId, len(activeTransactions))
		return len(activeTransactions)
	}, TIMEOUT, POLLING_INTERVAL).Should(Equal(0), "failed waiting for virtual guest to have no active transactions")
}

func SshKeyPresent(sshKeyId int) bool {
	accountService, err := CreateAccountService()
	Expect(err).ToNot(HaveOccurred())
	sshKeys, err := accountService.GetSshKeys()
	Expect(err).ToNot(HaveOccurred())

	for _, sshKey := range sshKeys {
		if sshKey.Id == sshKeyId {
			return true
		}
	}
	return false
}

func WaitForDeletedSshKeyToNoLongerBePresent(sshKeyId int) {
	accountService, err := CreateAccountService()
	Expect(err).ToNot(HaveOccurred())

	fmt.Printf("----> waiting for deleted ssh key to no longer be present\n")
	Eventually(func() bool {
		sshKeys, err := accountService.GetSshKeys()
		Expect(err).ToNot(HaveOccurred())

		for _, sshKey := range sshKeys {
			if sshKey.Id == sshKeyId {
				return false
			}
		}
		return true
	}, TIMEOUT, POLLING_INTERVAL).Should(BeTrue(), "failed waiting for deleted ssh key to be removed from list of ssh keys")
}

func WaitForCreatedSshKeyToBePresent(sshKeyId int) {
	accountService, err := CreateAccountService()
	Expect(err).ToNot(HaveOccurred())

	fmt.Printf("----> waiting for created ssh key to be present\n")
	Eventually(func() bool {
		sshKeys, err := accountService.GetSshKeys()
		Expect(err).ToNot(HaveOccurred())

		for _, sshKey := range sshKeys {
			if sshKey.Id == sshKeyId {
				return true
			}
		}
		return false
	}, TIMEOUT, POLLING_INTERVAL).Should(BeTrue(), "created ssh key but not in the list of ssh keys")
}

func WaitForVirtualGuestBlockTemplateGroupToHaveNoActiveTransactions(virtualGuestBlockTemplateGroupId int) {
	vgbdtgService, err := CreateVirtualGuestBlockDeviceTemplateGroupService()
	Expect(err).ToNot(HaveOccurred())

	fmt.Printf("----> waiting for virtual guest block template group to have no active transactions pending\n")
	Eventually(func() bool {
		activeTransaction, err := vgbdtgService.GetTransaction(virtualGuestBlockTemplateGroupId)
		Expect(err).ToNot(HaveOccurred())

		transactionTrue := false
		emptyTransaction := datatypes.SoftLayer_Provisioning_Version1_Transaction{}
		if activeTransaction != emptyTransaction {
			fmt.Printf("----> virtual guest template group: %d, has %#v pending\n", virtualGuestBlockTemplateGroupId, activeTransaction)
			transactionTrue = true
		}
		return transactionTrue
	}, TIMEOUT, POLLING_INTERVAL).Should(BeFalse(), "failed waiting for virtual guest block template group to have no active transactions")
}

func SetUserDataToVirtualGuest(virtualGuestId int, metadata string) {
	virtualGuestService, err := CreateVirtualGuestService()
	Expect(err).ToNot(HaveOccurred())

	success, err := virtualGuestService.SetMetadata(virtualGuestId, metadata)
	Expect(err).ToNot(HaveOccurred())
	Expect(success).To(BeTrue())
	fmt.Printf(fmt.Sprintf("----> successfully set metadata: `%s` to virtual guest instance: %d\n", metadata, virtualGuestId))
}

func ConfigureMetadataDiskOnVirtualGuest(virtualGuestId int) datatypes.SoftLayer_Provisioning_Version1_Transaction {
	virtualGuestService, err := CreateVirtualGuestService()
	Expect(err).ToNot(HaveOccurred())

	transaction, err := virtualGuestService.ConfigureMetadataDisk(virtualGuestId)
	Expect(err).ToNot(HaveOccurred())
	fmt.Printf(fmt.Sprintf("----> successfully configured metadata disk for virtual guest instance: %d\n", virtualGuestId))

	return transaction
}

func SetUserMetadataAndConfigureDisk(virtualGuestId int, userMetadata string) datatypes.SoftLayer_Provisioning_Version1_Transaction {
	SetUserDataToVirtualGuest(virtualGuestId, userMetadata)
	transaction := ConfigureMetadataDiskOnVirtualGuest(virtualGuestId)
	Expect(transaction.Id).ToNot(Equal(0))

	return transaction
}

func RunCommand(timeout time.Duration, cmd string, args ...string) *Session {
	command := exec.Command(cmd, args...)
	session, err := Start(command, GinkgoWriter, GinkgoWriter)
	Ω(err).ShouldNot(HaveOccurred())
	session.Wait(timeout)
	return session
}

func ScpToVirtualGuest(virtualGuestId int, sshKeyFilePath string, localFilePath string, remotePath string) {
	virtualGuestService, err := CreateVirtualGuestService()
	Expect(err).ToNot(HaveOccurred())

	virtualGuest, err := virtualGuestService.GetObject(virtualGuestId)
	Expect(err).ToNot(HaveOccurred())

	fmt.Printf("----> sending SCP command: %s\n", fmt.Sprintf("scp -i %s %s root@%s:%s", sshKeyFilePath, localFilePath, virtualGuest.PrimaryIpAddress, remotePath))
	session := RunCommand(TIMEOUT, "scp", "-o", "StrictHostKeyChecking=no", "-i", sshKeyFilePath, localFilePath, fmt.Sprintf("root@%s:%s", virtualGuest.PrimaryIpAddress, remotePath))
	Ω(session.ExitCode()).Should(Equal(0))
}

func SshExecOnVirtualGuest(virtualGuestId int, sshKeyFilePath string, remoteFilePath string, args ...string) int {
	virtualGuestService, err := CreateVirtualGuestService()
	Expect(err).ToNot(HaveOccurred())

	virtualGuest, err := virtualGuestService.GetObject(virtualGuestId)
	Expect(err).ToNot(HaveOccurred())

	fmt.Printf("----> sending SSH command: %s\n", fmt.Sprintf("ssh -i %s root@%s '%s \"%s\"'", sshKeyFilePath, virtualGuest.PrimaryIpAddress, remoteFilePath, args[0]))
	session := RunCommand(TIMEOUT, "ssh", "-o", "StrictHostKeyChecking=no", "-i", sshKeyFilePath, fmt.Sprintf("root@%s", virtualGuest.PrimaryIpAddress), fmt.Sprintf("%s \"%s\"", remoteFilePath, args[0]))

	return session.ExitCode()
}

func TestUserMetadata(userMetadata, sshKeyValue string) {
	workingDir, err := os.Getwd()
	Expect(err).ToNot(HaveOccurred())

	sshKeyFilePath := filepath.Join(workingDir, "sshTemp")
	err = ioutil.WriteFile(sshKeyFilePath, []byte(sshKeyValue), 600)
	defer os.Remove(sshKeyFilePath)
	Expect(err).ToNot(HaveOccurred())

	fetchUserMetadataShFilePath := filepath.Join(workingDir, "..", "scripts", "fetch_user_metadata.sh")
	Expect(err).ToNot(HaveOccurred())

	ScpToVirtualGuest(6396994, sshKeyFilePath, fetchUserMetadataShFilePath, "/tmp")
	retCode := SshExecOnVirtualGuest(6396994, sshKeyFilePath, "/tmp/fetch_user_metadata.sh", userMetadata)
	Expect(retCode).To(Equal(0))
}

func GetVirtualGuestPrimaryIpAddress(virtualGuestId int) string {
	virtualGuestService, err := CreateVirtualGuestService()
	Expect(err).ToNot(HaveOccurred())

	vgIpAddress, err := virtualGuestService.GetPrimaryIpAddress(virtualGuestId)
	Expect(err).ToNot(HaveOccurred())

	return vgIpAddress
}

func CreateDnsDomainService() (softlayer.SoftLayer_Dns_Domain_Service, error) {
	username, apiKey, err := GetUsernameAndApiKey()
	if err != nil {
		return nil, err
	}

	client := slclient.NewSoftLayerClient(username, apiKey)
	dnsDomainService, err := client.GetSoftLayer_Dns_Domain_Service()
	if err != nil {
		return nil, err
	}

	return dnsDomainService, nil
}

func CreateDnsDomainResourceRecordService() (softlayer.SoftLayer_Dns_Domain_ResourceRecord_Service, error) {
	username, apiKey, err := GetUsernameAndApiKey()
	if err != nil {
		return nil, err
	}

	client := slclient.NewSoftLayerClient(username, apiKey)
	dnsDomainResourceRecordService, err := client.GetSoftLayer_Dns_Domain_ResourceRecord_Service()
	if err != nil {
		return nil, err
	}

	return dnsDomainResourceRecordService, nil
}

func CreateTestDnsDomain(name string) datatypes.SoftLayer_Dns_Domain {
	template := datatypes.SoftLayer_Dns_Domain_Template{
		Name: name,
	}

	dnsDomainService, err := CreateDnsDomainService()
	Expect(err).ToNot(HaveOccurred())

	fmt.Printf("----> creating dns domain in SL\n")
	createdDnsDomain, err := dnsDomainService.CreateObject(template)
	Expect(err).ToNot(HaveOccurred())

	Expect(createdDnsDomain.Name).To(Equal(template.Name))
	fmt.Printf("----> created dns domain : %d\n in SL", createdDnsDomain.Id)

	return createdDnsDomain
}

func WaitForCreatedDnsDomainToBePresent(dnsDomainId int) {
	dnsDomainService, err := CreateDnsDomainService()
	Expect(err).ToNot(HaveOccurred())

	fmt.Printf("----> waiting for created dns domain to be present\n")
	Eventually(func() bool {
		dnsDomain, err := dnsDomainService.GetObject(dnsDomainId)
		Expect(err).ToNot(HaveOccurred())

		if dnsDomain.Id == dnsDomainId {
			return true
		}
		return false
	}, TIMEOUT, POLLING_INTERVAL).Should(BeTrue(), "created dns domain but not found")
}

func WaitForDeletedDnsDomainToNoLongerBePresent(dnsDomainId int) {
	dnsDomainService, err := CreateDnsDomainService()
	Expect(err).ToNot(HaveOccurred())

	fmt.Printf("----> waiting for deleted dns domain to no longer be present\n")
	Eventually(func() bool {
		dnsDomain, err := dnsDomainService.GetObject(dnsDomainId)
		Expect(err).ToNot(HaveOccurred())

		if dnsDomain.Id == dnsDomainId {
			return false
		}
		return true
	}, TIMEOUT, POLLING_INTERVAL).Should(BeTrue(), "failed waiting for deleted dns domain to be removed")
}

func CreateTestDnsDomainResourceRecord(domainId int) datatypes.SoftLayer_Dns_Domain_ResourceRecord {
	template := datatypes.SoftLayer_Dns_Domain_ResourceRecord_Template{
		Data:              "127.0.0.1",
		DomainId:          domainId,
		Host:              TEST_HOST,
		ResponsiblePerson: TEST_EMAIL,
		Ttl:               TEST_TTL,
		Type:              "A",
	}

	dnsDomainResourceRecordService, err := CreateDnsDomainResourceRecordService()
	Expect(err).ToNot(HaveOccurred())

	fmt.Printf("----> creating dns domain resource record in SL\n")
	createdDnsDomainResourceRecord, err := dnsDomainResourceRecordService.CreateObject(template)
	Expect(err).ToNot(HaveOccurred())

	Expect(createdDnsDomainResourceRecord.Data).To(Equal(template.Data), "127.0.0.1")
	Expect(createdDnsDomainResourceRecord.Host).To(Equal(template.Host), TEST_HOST)
	Expect(createdDnsDomainResourceRecord.ResponsiblePerson).To(Equal(template.ResponsiblePerson), TEST_EMAIL)
	Expect(createdDnsDomainResourceRecord.Ttl).To(Equal(template.Ttl), "900")
	Expect(createdDnsDomainResourceRecord.Type).To(Equal(template.Type), "A")
	fmt.Printf("----> created dns domain resource record: %d\n in SL", createdDnsDomainResourceRecord.Id)

	return createdDnsDomainResourceRecord
}

func WaitForCreatedDnsDomainResourceRecordToBePresent(dnsDomainResourceRecordId int) {
	dnsDomainResourceRecordService, err := CreateDnsDomainResourceRecordService()
	Expect(err).ToNot(HaveOccurred())

	fmt.Printf("----> waiting for created dns domain resource record to be present\n")
	Eventually(func() bool {
		dnsDomainResourceRecord, err := dnsDomainResourceRecordService.GetObject(dnsDomainResourceRecordId)
		Expect(err).ToNot(HaveOccurred())

		if dnsDomainResourceRecord.Id == dnsDomainResourceRecordId {
			return true
		}
		return false
	}, TIMEOUT, POLLING_INTERVAL).Should(BeTrue(), "created dns domain resource record but not found")
}

func WaitForDeletedDnsDomainResourceRecordToNoLongerBePresent(dnsDomainResourceRecordId int) {
	dnsDomainResourceRecordService, err := CreateDnsDomainResourceRecordService()
	Expect(err).ToNot(HaveOccurred())

	fmt.Printf("----> waiting for deleted dns domain resource record to no longer be present\n")
	Eventually(func() bool {
		dnsDomainResourceRecord, err := dnsDomainResourceRecordService.GetObject(dnsDomainResourceRecordId)
		Expect(err).ToNot(HaveOccurred())

		if dnsDomainResourceRecord.Id == dnsDomainResourceRecordId {
			return false
		}
		return true
	}, TIMEOUT, POLLING_INTERVAL).Should(BeTrue(), "failed waiting for deleted dns domain resource record to be removed")
}

// Private functions

func generateSshKeyUsingGo() (string, string, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2014)
	if err != nil {
		return "", "", err
	}

	fmt.Printf("----> creating ssh private key using Golang\n")
	privateKeyDer := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyBlock := pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   privateKeyDer,
	}
	privateKeyPem := string(pem.EncodeToMemory(&privateKeyBlock))

	fmt.Printf("----> creating ssh public key using Golang\n")
	publicKey := privateKey.PublicKey
	publicKeyDer, err := x509.MarshalPKIXPublicKey(&publicKey)
	if err != nil {
		return "", "", err
	}

	publicKeyBlock := pem.Block{
		Type:    "PUBLIC KEY",
		Headers: nil,
		Bytes:   publicKeyDer,
	}

	publicKeyPem := string(pem.EncodeToMemory(&publicKeyBlock))

	return privateKeyPem, publicKeyPem, nil
}

func generateSshKeyUsingSshKeyGen() (string, string, error) {
	tmpDir, err := ioutil.TempDir("", "generateSshKeyUsingSshKeyGen")
	if err != nil {
		return "", "", err
	}

	rsaKeyFileName := filepath.Join(tmpDir, "ssh_key_file.rsa")
	rsaKeyFileNamePub := rsaKeyFileName + ".pub"

	sshKeyGen, err := exec.LookPath("ssh-keygen")
	if err != nil {
		return "", "", err
	}

	_, err = exec.Command(sshKeyGen,
		"-f", rsaKeyFileName,
		"-t", "rsa", "-N", "").Output()
	if err != nil {
		return "", "", err
	}

	privateKey, err := ioutil.ReadFile(rsaKeyFileName)
	if err != nil {
		return "", "", err
	}

	publicKey, err := ioutil.ReadFile(rsaKeyFileNamePub)
	if err != nil {
		return "", "", err
	}

	err = os.RemoveAll(tmpDir)
	if err != nil {
		return "", "", err
	}

	return string(privateKey), string(publicKey), nil
}
