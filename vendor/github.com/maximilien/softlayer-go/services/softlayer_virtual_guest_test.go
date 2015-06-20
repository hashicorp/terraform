package services_test

import (
	"errors"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	slclientfakes "github.com/maximilien/softlayer-go/client/fakes"
	datatypes "github.com/maximilien/softlayer-go/data_types"
	softlayer "github.com/maximilien/softlayer-go/softlayer"
	testhelpers "github.com/maximilien/softlayer-go/test_helpers"
)

var _ = Describe("SoftLayer_Virtual_Guest_Service", func() {

	var (
		username, apiKey string
		err              error

		fakeClient *slclientfakes.FakeSoftLayerClient

		virtualGuestService softlayer.SoftLayer_Virtual_Guest_Service

		virtualGuest         datatypes.SoftLayer_Virtual_Guest
		virtualGuestTemplate datatypes.SoftLayer_Virtual_Guest_Template
		reload_OS_Config     datatypes.Image_Template_Config
	)

	BeforeEach(func() {
		username = os.Getenv("SL_USERNAME")
		Expect(username).ToNot(Equal(""))

		apiKey = os.Getenv("SL_API_KEY")
		Expect(apiKey).ToNot(Equal(""))

		fakeClient = slclientfakes.NewFakeSoftLayerClient(username, apiKey)
		Expect(fakeClient).ToNot(BeNil())

		fakeClient.SoftLayerServices["SoftLayer_Product_Package"] = &testhelpers.MockProductPackageService{}

		virtualGuestService, err = fakeClient.GetSoftLayer_Virtual_Guest_Service()
		Expect(err).ToNot(HaveOccurred())
		Expect(virtualGuestService).ToNot(BeNil())

		virtualGuest = datatypes.SoftLayer_Virtual_Guest{}
		virtualGuestTemplate = datatypes.SoftLayer_Virtual_Guest_Template{}
	})

	Context("#GetName", func() {
		It("returns the name for the service", func() {
			name := virtualGuestService.GetName()
			Expect(name).To(Equal("SoftLayer_Virtual_Guest"))
		})
	})

	Context("#CreateObject", func() {
		BeforeEach(func() {
			fakeClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Virtual_Guest_Service_createObject.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("creates a new SoftLayer_Virtual_Guest instance", func() {
			virtualGuestTemplate = datatypes.SoftLayer_Virtual_Guest_Template{
				Hostname:  "fake-hostname",
				Domain:    "fake.domain.com",
				StartCpus: 2,
				MaxMemory: 1024,
				Datacenter: datatypes.Datacenter{
					Name: "fake-datacenter-name",
				},
				HourlyBillingFlag:            true,
				LocalDiskFlag:                false,
				DedicatedAccountHostOnlyFlag: false,
				NetworkComponents: []datatypes.NetworkComponents{datatypes.NetworkComponents{
					MaxSpeed: 10,
				}},
				UserData: []datatypes.UserData{
					datatypes.UserData{
						Value: "some user data $_/<| with special characters",
					},
				},
			}
			virtualGuest, err = virtualGuestService.CreateObject(virtualGuestTemplate)
			Expect(err).ToNot(HaveOccurred())
			Expect(virtualGuest.Hostname).To(Equal("fake-hostname"))
			Expect(virtualGuest.Domain).To(Equal("fake.domain.com"))
			Expect(virtualGuest.StartCpus).To(Equal(2))
			Expect(virtualGuest.MaxMemory).To(Equal(1024))
			Expect(virtualGuest.DedicatedAccountHostOnlyFlag).To(BeFalse())
		})

		It("flags all missing required parameters for SoftLayer_Virtual_Guest/createObject.json POST call", func() {
			virtualGuestTemplate = datatypes.SoftLayer_Virtual_Guest_Template{}
			_, err := virtualGuestService.CreateObject(virtualGuestTemplate)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Hostname"))
			Expect(err.Error()).To(ContainSubstring("Domain"))
			Expect(err.Error()).To(ContainSubstring("StartCpus"))
			Expect(err.Error()).To(ContainSubstring("MaxMemory"))
			Expect(err.Error()).To(ContainSubstring("Datacenter"))
		})
	})

	Context("#GetObject", func() {
		BeforeEach(func() {
			virtualGuest.Id = 1234567
			fakeClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Virtual_Guest_Service_getObject.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("sucessfully retrieves SoftLayer_Virtual_Guest instance", func() {
			vg, err := virtualGuestService.GetObject(virtualGuest.Id)
			Expect(err).ToNot(HaveOccurred())
			Expect(vg.Id).To(Equal(virtualGuest.Id))
			Expect(vg.AccountId).To(Equal(278444))
			Expect(vg.CreateDate).ToNot(BeNil())
			Expect(vg.DedicatedAccountHostOnlyFlag).To(BeFalse())
			Expect(vg.Domain).To(Equal("softlayer.com"))
			Expect(vg.FullyQualifiedDomainName).To(Equal("bosh-ecpi1.softlayer.com"))
			Expect(vg.Hostname).To(Equal("bosh-ecpi1"))
			Expect(vg.Id).To(Equal(1234567))
			Expect(vg.LastPowerStateId).To(Equal(0))
			Expect(vg.LastVerifiedDate).To(BeNil())
			Expect(vg.MaxCpu).To(Equal(1))
			Expect(vg.MaxCpuUnits).To(Equal("CORE"))
			Expect(vg.MaxMemory).To(Equal(1024))
			Expect(vg.MetricPollDate).To(BeNil())
			Expect(vg.ModifyDate).ToNot(BeNil())
			Expect(vg.StartCpus).To(Equal(1))
			Expect(vg.StatusId).To(Equal(1001))
			Expect(vg.Uuid).To(Equal("85d444ce-55a0-39c0-e17a-f697f223cd8a"))
			Expect(vg.GlobalIdentifier).To(Equal("52145e01-97b6-4312-9c15-dac7f24b6c2a"))
			Expect(vg.UserData[0].Value).To(Equal("some user data $_/<| with special characters"))
			Expect(vg.PrimaryBackendIpAddress).To(Equal("10.106.192.42"))
			Expect(vg.PrimaryIpAddress).To(Equal("23.246.234.32"))
			Expect(vg.Location.Id).To(Equal(1234567))
			Expect(vg.Location.Name).To(Equal("R5"))
			Expect(vg.Location.LongName).To(Equal("Room 5"))
			Expect(vg.Datacenter.Id).To(Equal(456))
			Expect(vg.Datacenter.Name).To(Equal("bej2"))
			Expect(vg.Datacenter.LongName).To(Equal("Beijing 2"))
			Expect(vg.NetworkComponents[0].MaxSpeed).To(Equal(100))
			Expect(len(vg.OperatingSystem.Passwords)).To(BeNumerically(">=", 1))
			Expect(vg.OperatingSystem.Passwords[0].Password).To(Equal("test_password"))
			Expect(vg.OperatingSystem.Passwords[0].Username).To(Equal("test_username"))
		})
	})

	Context("#EditObject", func() {
		BeforeEach(func() {
			virtualGuest.Id = 1234567
			fakeClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Virtual_Guest_Service_editObject.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("edits an existing SoftLayer_Virtual_Guest instance", func() {
			virtualGuest := datatypes.SoftLayer_Virtual_Guest{
				Notes: "fake-notes",
			}
			edited, err := virtualGuestService.EditObject(virtualGuest.Id, virtualGuest)
			Expect(err).ToNot(HaveOccurred())
			Expect(edited).To(BeTrue())
		})
	})

	Context("#ReloadOperatingSystem", func() {
		BeforeEach(func() {
			reload_OS_Config = datatypes.Image_Template_Config{
				ImageTemplateId: "5b7bc66a-72c6-447a-94a1-967803fcd76b",
			}
			virtualGuest.Id = 1234567
		})

		It("sucessfully reload OS on the virtual guest instance", func() {
			fakeClient.DoRawHttpRequestResponse = []byte(`"1"`)

			err = virtualGuestService.ReloadOperatingSystem(virtualGuest.Id, reload_OS_Config)
			Expect(err).ToNot(HaveOccurred())
		})

		It("fails to reload OS on the virtual guest instance", func() {
			fakeClient.DoRawHttpRequestResponse = []byte(`"99"`)

			err = virtualGuestService.ReloadOperatingSystem(virtualGuest.Id, reload_OS_Config)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("#DeleteObject", func() {
		BeforeEach(func() {
			virtualGuest.Id = 1234567
		})

		It("sucessfully deletes the SoftLayer_Virtual_Guest instance", func() {
			fakeClient.DoRawHttpRequestResponse = []byte("true")
			deleted, err := virtualGuestService.DeleteObject(virtualGuest.Id)
			Expect(err).ToNot(HaveOccurred())
			Expect(deleted).To(BeTrue())
		})

		It("fails to delete the SoftLayer_Virtual_Guest instance", func() {
			fakeClient.DoRawHttpRequestResponse = []byte("false")
			deleted, err := virtualGuestService.DeleteObject(virtualGuest.Id)
			Expect(err).To(HaveOccurred())
			Expect(deleted).To(BeFalse())
		})
	})

	Context("#AttachEphemeralDisk", func() {
		BeforeEach(func() {
			fakeClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Product_Order_placeOrder.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("reports error when providing a wrong disk size", func() {
			_, err := virtualGuestService.AttachEphemeralDisk(123, -1)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Ephemeral disk size can not be negative: -1"))
		})

		It("can attach a local disk without error", func() {
			receipt, err := virtualGuestService.AttachEphemeralDisk(123, 25)
			Expect(err).ToNot(HaveOccurred())
			Expect(receipt.OrderId).NotTo(Equal(0))
		})

	})

	Context("#UpgradeObject", func() {
		BeforeEach(func() {
			fakeClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Product_Order_placeOrder.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("can upgrade object without any error", func() {
			_, err := virtualGuestService.UpgradeObject(123, &softlayer.UpgradeOptions{
				Cpus:       2,
				MemoryInGB: 2,
				NicSpeed:   1000,
			})
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("#GetAvailableUpgradeItemPrices", func() {
		BeforeEach(func() {
			fakeClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Product_Order_placeOrder.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("reports error when pricing item for provided CPUs is not available", func() {
			_, err := virtualGuestService.GetAvailableUpgradeItemPrices(&softlayer.UpgradeOptions{Cpus: 3})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Failed to find price for 'cpus' (of size 3)"))
		})

		It("reports error when pricing item for provided RAM is not available", func() {
			_, err := virtualGuestService.GetAvailableUpgradeItemPrices(&softlayer.UpgradeOptions{MemoryInGB: 1500})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Failed to find price for 'memory' (of size 1500)"))
		})

		It("reports error when pricing item for provided network speed is not available", func() {
			_, err := virtualGuestService.GetAvailableUpgradeItemPrices(&softlayer.UpgradeOptions{NicSpeed: 999})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Failed to find price for 'nic_speed' (of size 999)"))
		})
	})

	Context("#GetPowerState", func() {
		BeforeEach(func() {
			virtualGuest.Id = 1234567
			fakeClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Virtual_Guest_Service_getPowerState.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("sucessfully retrieves SoftLayer_Virtual_Guest_State for RUNNING instance", func() {
			vgPowerState, err := virtualGuestService.GetPowerState(virtualGuest.Id)
			Expect(err).ToNot(HaveOccurred())
			Expect(vgPowerState.KeyName).To(Equal("RUNNING"))
		})
	})

	Context("#GetPrimaryIpAddress", func() {
		BeforeEach(func() {
			virtualGuest.Id = 1234567
			fakeClient.DoRawHttpRequestResponse = []byte("159.99.99.99")
			Expect(err).ToNot(HaveOccurred())
		})

		It("sucessfully retrieves SoftLayer virtual guest's primary IP address instance", func() {
			vgPrimaryIpAddress, err := virtualGuestService.GetPrimaryIpAddress(virtualGuest.Id)
			Expect(err).ToNot(HaveOccurred())
			Expect(vgPrimaryIpAddress).To(Equal("159.99.99.99"))
		})
	})

	Context("#GetActiveTransaction", func() {
		BeforeEach(func() {
			virtualGuest.Id = 1234567
			fakeClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Virtual_Guest_Service_getActiveTransaction.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("sucessfully retrieves SoftLayer_Provisioning_Version1_Transaction for virtual guest", func() {
			activeTransaction, err := virtualGuestService.GetActiveTransaction(virtualGuest.Id)
			Expect(err).ToNot(HaveOccurred())
			Expect(activeTransaction.CreateDate).ToNot(BeNil())
			Expect(activeTransaction.ElapsedSeconds).To(BeNumerically(">", 0))
			Expect(activeTransaction.GuestId).To(Equal(virtualGuest.Id))
			Expect(activeTransaction.Id).To(BeNumerically(">", 0))
		})
	})

	Context("#GetLastTransaction", func() {
		BeforeEach(func() {
			virtualGuest.Id = 1234567
			fakeClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Virtual_Guest_Service_getLastTransaction.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("sucessfully retrieves last SoftLayer_Provisioning_Version1_Transaction for virtual guest", func() {
			lastTransaction, err := virtualGuestService.GetLastTransaction(virtualGuest.Id)
			Expect(err).ToNot(HaveOccurred())
			Expect(lastTransaction.CreateDate).ToNot(BeNil())
			Expect(lastTransaction.ElapsedSeconds).To(BeNumerically(">", 0))
			Expect(lastTransaction.GuestId).To(Equal(virtualGuest.Id))
			Expect(lastTransaction.Id).To(BeNumerically(">", 0))
		})
	})

	Context("#GetActiveTransactions", func() {
		BeforeEach(func() {
			virtualGuest.Id = 1234567
			fakeClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Virtual_Guest_Service_getActiveTransactions.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("sucessfully retrieves an array of SoftLayer_Provisioning_Version1_Transaction for virtual guest", func() {
			activeTransactions, err := virtualGuestService.GetActiveTransactions(virtualGuest.Id)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(activeTransactions)).To(BeNumerically(">", 0))

			for _, activeTransaction := range activeTransactions {
				Expect(activeTransaction.CreateDate).ToNot(BeNil())
				Expect(activeTransaction.ElapsedSeconds).To(BeNumerically(">", 0))
				Expect(activeTransaction.GuestId).To(Equal(virtualGuest.Id))
				Expect(activeTransaction.Id).To(BeNumerically(">", 0))
			}
		})
	})

	Context("#GetSshKeys", func() {
		BeforeEach(func() {
			virtualGuest.Id = 1234567
			fakeClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Virtual_Guest_Service_getSshKeys.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("sucessfully retrieves an array of SoftLayer_Security_Ssh_Key for virtual guest", func() {
			sshKeys, err := virtualGuestService.GetSshKeys(virtualGuest.Id)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(sshKeys)).To(BeNumerically(">", 0))

			for _, sshKey := range sshKeys {
				Expect(sshKey.CreateDate).ToNot(BeNil())
				Expect(sshKey.Fingerprint).To(Equal("f6:c2:9d:57:2f:74:be:a1:db:71:f2:e5:8e:0f:84:7e"))
				Expect(sshKey.Id).To(Equal(84386))
				Expect(sshKey.Key).ToNot(Equal(""))
				Expect(sshKey.Label).To(Equal("TEST:softlayer-go"))
				Expect(sshKey.ModifyDate).To(BeNil())
				Expect(sshKey.Label).To(Equal("TEST:softlayer-go"))
			}
		})
	})

	Context("#PowerCycle", func() {
		BeforeEach(func() {
			virtualGuest.Id = 1234567
		})

		It("sucessfully power cycle virtual guest instance", func() {
			fakeClient.DoRawHttpRequestResponse = []byte("true")

			rebooted, err := virtualGuestService.PowerCycle(virtualGuest.Id)
			Expect(err).ToNot(HaveOccurred())
			Expect(rebooted).To(BeTrue())
		})

		It("fails to power cycle virtual guest instance", func() {
			fakeClient.DoRawHttpRequestResponse = []byte("false")

			rebooted, err := virtualGuestService.PowerCycle(virtualGuest.Id)
			Expect(err).To(HaveOccurred())
			Expect(rebooted).To(BeFalse())
		})
	})

	Context("#PowerOff", func() {
		BeforeEach(func() {
			virtualGuest.Id = 1234567
		})

		It("sucessfully power off virtual guest instance", func() {
			fakeClient.DoRawHttpRequestResponse = []byte("true")

			rebooted, err := virtualGuestService.PowerOff(virtualGuest.Id)
			Expect(err).ToNot(HaveOccurred())
			Expect(rebooted).To(BeTrue())
		})

		It("fails to power off virtual guest instance", func() {
			fakeClient.DoRawHttpRequestResponse = []byte("false")

			rebooted, err := virtualGuestService.PowerOff(virtualGuest.Id)
			Expect(err).To(HaveOccurred())
			Expect(rebooted).To(BeFalse())
		})
	})

	Context("#PowerOffSoft", func() {
		BeforeEach(func() {
			virtualGuest.Id = 1234567
		})

		It("sucessfully power off soft virtual guest instance", func() {
			fakeClient.DoRawHttpRequestResponse = []byte("true")

			rebooted, err := virtualGuestService.PowerOffSoft(virtualGuest.Id)
			Expect(err).ToNot(HaveOccurred())
			Expect(rebooted).To(BeTrue())
		})

		It("fails to power off soft virtual guest instance", func() {
			fakeClient.DoRawHttpRequestResponse = []byte("false")

			rebooted, err := virtualGuestService.PowerOffSoft(virtualGuest.Id)
			Expect(err).To(HaveOccurred())
			Expect(rebooted).To(BeFalse())
		})
	})

	Context("#PowerOn", func() {
		BeforeEach(func() {
			virtualGuest.Id = 1234567
		})

		It("sucessfully power on virtual guest instance", func() {
			fakeClient.DoRawHttpRequestResponse = []byte("true")

			rebooted, err := virtualGuestService.PowerOn(virtualGuest.Id)
			Expect(err).ToNot(HaveOccurred())
			Expect(rebooted).To(BeTrue())
		})

		It("fails to power on virtual guest instance", func() {
			fakeClient.DoRawHttpRequestResponse = []byte("false")

			rebooted, err := virtualGuestService.PowerOn(virtualGuest.Id)
			Expect(err).To(HaveOccurred())
			Expect(rebooted).To(BeFalse())
		})
	})

	Context("#RebootDefault", func() {
		BeforeEach(func() {
			virtualGuest.Id = 1234567
		})

		It("sucessfully default reboots virtual guest instance", func() {
			fakeClient.DoRawHttpRequestResponse = []byte("true")

			rebooted, err := virtualGuestService.RebootDefault(virtualGuest.Id)
			Expect(err).ToNot(HaveOccurred())
			Expect(rebooted).To(BeTrue())
		})

		It("fails to default reboot virtual guest instance", func() {
			fakeClient.DoRawHttpRequestResponse = []byte("false")

			rebooted, err := virtualGuestService.RebootDefault(virtualGuest.Id)
			Expect(err).To(HaveOccurred())
			Expect(rebooted).To(BeFalse())
		})
	})

	Context("#RebootSoft", func() {
		BeforeEach(func() {
			virtualGuest.Id = 1234567
		})

		It("sucessfully soft reboots virtual guest instance", func() {
			fakeClient.DoRawHttpRequestResponse = []byte("true")

			rebooted, err := virtualGuestService.RebootSoft(virtualGuest.Id)
			Expect(err).ToNot(HaveOccurred())
			Expect(rebooted).To(BeTrue())
		})

		It("fails to soft reboot virtual guest instance", func() {
			fakeClient.DoRawHttpRequestResponse = []byte("false")

			rebooted, err := virtualGuestService.RebootSoft(virtualGuest.Id)
			Expect(err).To(HaveOccurred())
			Expect(rebooted).To(BeFalse())
		})
	})

	Context("#RebootHard", func() {
		BeforeEach(func() {
			virtualGuest.Id = 1234567
		})

		It("sucessfully hard reboot virtual guest instance", func() {
			fakeClient.DoRawHttpRequestResponse = []byte("true")

			rebooted, err := virtualGuestService.RebootHard(virtualGuest.Id)
			Expect(err).ToNot(HaveOccurred())
			Expect(rebooted).To(BeTrue())
		})

		It("fails to hard reboot virtual guest instance", func() {
			fakeClient.DoRawHttpRequestResponse = []byte("false")

			rebooted, err := virtualGuestService.RebootHard(virtualGuest.Id)
			Expect(err).To(HaveOccurred())
			Expect(rebooted).To(BeFalse())
		})
	})

	Context("#SetUserMetadata", func() {
		BeforeEach(func() {
			virtualGuest.Id = 1234567
			fakeClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Virtual_Guest_Service_setMetadata.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("sucessfully adds metadata strings as a dile to virtual guest's metadata disk", func() {
			retBool, err := virtualGuestService.SetMetadata(virtualGuest.Id, "fake-metadata")
			Expect(err).ToNot(HaveOccurred())

			Expect(retBool).To(BeTrue())
		})
	})

	Context("#GetUserData", func() {
		BeforeEach(func() {
			virtualGuest.Id = 1234567
			fakeClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Virtual_Guest_Service_getUserData.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("sucessfully returns user data for the virtual guest", func() {
			attributes, err := virtualGuestService.GetUserData(virtualGuest.Id)
			Expect(err).ToNot(HaveOccurred())

			Expect(len(attributes)).To(BeNumerically("==", 2))

			Expect(attributes[0].Value).To(Equal("V2hvJ3Mgc21hcnRlcj8gRG1pdHJ5aSBvciBkci5tYXguLi4gIHRoZSBkb2MsIGFueSBkYXkgOik="))
			Expect(attributes[0].Type.Name).To(Equal("User Data"))
			Expect(attributes[0].Type.Keyname).To(Equal("USER_DATA"))

			Expect(attributes[1].Value).To(Equal("ZmFrZS1iYXNlNjQtZGF0YQo="))
			Expect(attributes[1].Type.Name).To(Equal("Fake Data"))
			Expect(attributes[1].Type.Keyname).To(Equal("FAKE_DATA"))
		})
	})

	Context("#IsPingable", func() {
		BeforeEach(func() {
			virtualGuest.Id = 1234567
		})

		Context("when there are no API errors", func() {
			It("checks that the virtual guest instance is pigable", func() {
				fakeClient.DoRawHttpRequestResponse = []byte("true")

				pingable, err := virtualGuestService.IsPingable(virtualGuest.Id)
				Expect(err).ToNot(HaveOccurred())
				Expect(pingable).To(BeTrue())
			})

			It("checks that the virtual guest instance is NOT pigable", func() {
				fakeClient.DoRawHttpRequestResponse = []byte("false")

				pingable, err := virtualGuestService.IsPingable(virtualGuest.Id)
				Expect(err).ToNot(HaveOccurred())
				Expect(pingable).To(BeFalse())
			})
		})

		Context("when there are API errors", func() {
			It("returns false and error", func() {
				fakeClient.DoRawHttpRequestError = errors.New("fake-error")

				pingable, err := virtualGuestService.IsPingable(virtualGuest.Id)
				Expect(err).To(HaveOccurred())
				Expect(pingable).To(BeFalse())
			})
		})

		Context("when the API returns invalid or empty result", func() {
			It("returns false and error", func() {
				fakeClient.DoRawHttpRequestResponse = []byte("fake")

				pingable, err := virtualGuestService.IsPingable(virtualGuest.Id)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Failed to checking that virtual guest is pingable"))
				Expect(pingable).To(BeFalse())
			})
		})
	})

	Context("#IsBackendPingeable", func() {
		BeforeEach(func() {
			virtualGuest.Id = 1234567
		})

		Context("when there are no API errors", func() {
			It("checks that the virtual guest instance backend is pigable", func() {
				fakeClient.DoRawHttpRequestResponse = []byte("true")

				pingable, err := virtualGuestService.IsBackendPingable(virtualGuest.Id)
				Expect(err).ToNot(HaveOccurred())
				Expect(pingable).To(BeTrue())
			})

			It("checks that the virtual guest instance backend is NOT pigable", func() {
				fakeClient.DoRawHttpRequestResponse = []byte("false")

				pingable, err := virtualGuestService.IsBackendPingable(virtualGuest.Id)
				Expect(err).ToNot(HaveOccurred())
				Expect(pingable).To(BeFalse())
			})
		})

		Context("when there are API errors", func() {
			It("returns false and error", func() {
				fakeClient.DoRawHttpRequestError = errors.New("fake-error")

				pingable, err := virtualGuestService.IsBackendPingable(virtualGuest.Id)
				Expect(err).To(HaveOccurred())
				Expect(pingable).To(BeFalse())
			})
		})

		Context("when the API returns invalid or empty result", func() {
			It("returns false and error", func() {
				fakeClient.DoRawHttpRequestResponse = []byte("fake")

				pingable, err := virtualGuestService.IsBackendPingable(virtualGuest.Id)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Failed to checking that virtual guest backend is pingable"))
				Expect(pingable).To(BeFalse())
			})
		})
	})

	Context("#ConfigureMetadataDisk", func() {
		BeforeEach(func() {
			virtualGuest.Id = 1234567
			fakeClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Virtual_Guest_Service_configureMetadataDisk.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("sucessfully configures a metadata disk for a virtual guest", func() {
			transaction, err := virtualGuestService.ConfigureMetadataDisk(virtualGuest.Id)
			Expect(err).ToNot(HaveOccurred())

			Expect(transaction.CreateDate).ToNot(BeNil())
			Expect(transaction.ElapsedSeconds).To(Equal(0))
			Expect(transaction.GuestId).To(Equal(virtualGuest.Id))
			Expect(transaction.HardwareId).To(Equal(0))
			Expect(transaction.Id).To(Equal(12476326))
			Expect(transaction.ModifyDate).ToNot(BeNil())
			Expect(transaction.StatusChangeDate).ToNot(BeNil())

			Expect(transaction.TransactionGroup.AverageTimeToComplete).To(Equal("1.62"))
			Expect(transaction.TransactionGroup.Name).To(Equal("Configure Cloud Metadata Disk"))

			Expect(transaction.TransactionStatus.AverageDuration).To(Equal(".32"))
			Expect(transaction.TransactionStatus.FriendlyName).To(Equal("Configure Cloud Metadata Disk"))
			Expect(transaction.TransactionStatus.Name).To(Equal("CLOUD_CONFIGURE_METADATA_DISK"))
		})
	})

	Context("#GetUpgradeItemPrices", func() {
		BeforeEach(func() {
			virtualGuest.Id = 1234567
			fakeClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Virtual_Guest_Service_getUpgradeItemPrices.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("sucessfully get the upgrade item prices for a virtual guest", func() {
			itemPrices, err := virtualGuestService.GetUpgradeItemPrices(virtualGuest.Id)
			Expect(err).ToNot(HaveOccurred())

			Expect(len(itemPrices)).To(Equal(1))
			Expect(itemPrices[0].Id).To(Equal(12345))
			Expect(itemPrices[0].Categories[0].CategoryCode).To(Equal("guest_disk1"))
		})
	})

	Context("#SetTags", func() {
		BeforeEach(func() {
			virtualGuest.Id = 1234567
			fakeClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Virtual_Guest_Service_setTags.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("sets tags: tag0, tag1, tag2 to virtual guest instance", func() {
			tags := []string{"tag0", "tag1", "tag2"}
			tagsWasSet, err := virtualGuestService.SetTags(virtualGuest.Id, tags)

			Expect(err).ToNot(HaveOccurred())
			Expect(tagsWasSet).To(BeTrue())
		})
	})

	Context("#GetReferenceTags", func() {
		BeforeEach(func() {
			virtualGuest.Id = 1234567
			fakeClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Virtual_Guest_Service_getReferenceTags.json")
			Expect(err).ToNot(HaveOccurred())
		})

		itVerifiesATagReference := func(tagReference1 datatypes.SoftLayer_Tag_Reference, tagReference2 datatypes.SoftLayer_Tag_Reference) {
			Expect(tagReference1.EmpRecordId).To(Equal(tagReference2.EmpRecordId))
			Expect(tagReference1.Id).To(Equal(tagReference2.Id))
			Expect(tagReference1.ResourceTableId).To(Equal(tagReference2.ResourceTableId))

			Expect(tagReference1.Tag.AccountId).To(Equal(tagReference2.Tag.AccountId))
			Expect(tagReference1.Tag.Id).To(Equal(tagReference2.Tag.Id))
			Expect(tagReference1.Tag.Internal).To(Equal(tagReference2.Tag.Internal))
			Expect(tagReference1.Tag.Name).To(Equal(tagReference2.Tag.Name))

			Expect(tagReference1.TagId).To(Equal(tagReference2.TagId))

			Expect(tagReference1.TagType.Description).To(Equal(tagReference2.TagType.Description))
			Expect(tagReference1.TagType.KeyName).To(Equal(tagReference2.TagType.KeyName))

			Expect(tagReference1.TagTypeId).To(Equal(tagReference2.TagTypeId))
			Expect(tagReference1.UsrRecordId).To(Equal(tagReference2.UsrRecordId))
		}

		It("gets the reference tags: tag0, tag1, tag2 from the virtual guest instance", func() {
			tagReferences, err := virtualGuestService.GetTagReferences(virtualGuest.Id)

			Expect(err).ToNot(HaveOccurred())
			Expect(len(tagReferences)).To(Equal(3))

			expectedTagReferences := []datatypes.SoftLayer_Tag_Reference{
				datatypes.SoftLayer_Tag_Reference{
					EmpRecordId:     nil,
					Id:              1855150,
					ResourceTableId: 7967498,
					Tag: datatypes.TagReference{
						AccountId: 278444,
						Id:        91128,
						Internal:  0,
						Name:      "tag1",
					},
					TagId: 91128,
					TagType: datatypes.TagType{
						Description: "CCI",
						KeyName:     "GUEST",
					},
					TagTypeId:   2,
					UsrRecordId: 239954,
				},
				datatypes.SoftLayer_Tag_Reference{
					EmpRecordId:     nil,
					Id:              1855152,
					ResourceTableId: 7967498,
					Tag: datatypes.TagReference{
						AccountId: 278444,
						Id:        91130,
						Internal:  0,
						Name:      "tag2",
					},
					TagId: 91130,
					TagType: datatypes.TagType{
						Description: "CCI",
						KeyName:     "GUEST",
					},
					TagTypeId:   2,
					UsrRecordId: 239954,
				},
				datatypes.SoftLayer_Tag_Reference{
					EmpRecordId:     nil,
					Id:              1855154,
					ResourceTableId: 7967498,
					Tag: datatypes.TagReference{
						AccountId: 278444,
						Id:        91132,
						Internal:  0,
						Name:      "tag3",
					},
					TagId: 91132,
					TagType: datatypes.TagType{
						Description: "CCI",
						KeyName:     "GUEST",
					},
					TagTypeId:   2,
					UsrRecordId: 239954,
				},
			}
			for i, expectedTagReference := range expectedTagReferences {
				itVerifiesATagReference(tagReferences[i], expectedTagReference)
			}
		})
	})

	Context("#AttachDiskImage", func() {
		BeforeEach(func() {
			virtualGuest.Id = 1234567
			fakeClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Virtual_Guest_Service_attachDiskImage.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("attaches disk image with ID `1234567` to virtual guest instance", func() {
			imageId := 1234567
			transaction, err := virtualGuestService.AttachDiskImage(virtualGuest.Id, imageId)

			Expect(err).ToNot(HaveOccurred())
			Expect(transaction).ToNot(Equal(datatypes.SoftLayer_Provisioning_Version1_Transaction{}))
		})
	})

	Context("#DetachDiskImage", func() {
		BeforeEach(func() {
			virtualGuest.Id = 1234567
			fakeClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Virtual_Guest_Service_detachDiskImage.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("detaches disk image with ID `1234567` to virtual guest instance", func() {
			imageId := 1234567
			transaction, err := virtualGuestService.DetachDiskImage(virtualGuest.Id, imageId)

			Expect(err).ToNot(HaveOccurred())
			Expect(transaction).ToNot(Equal(datatypes.SoftLayer_Provisioning_Version1_Transaction{}))
		})
	})

	Context("#ActivatePrivatePort", func() {
		BeforeEach(func() {
			virtualGuest.Id = 1234567
			fakeClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Virtual_Guest_Service_activatePrivatePort.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("activates private port for virtual guest instance", func() {
			activated, err := virtualGuestService.ActivatePrivatePort(virtualGuest.Id)

			Expect(err).ToNot(HaveOccurred())
			Expect(activated).To(BeTrue())
		})
	})

	Context("#ActivatePublicPort", func() {
		BeforeEach(func() {
			virtualGuest.Id = 1234567
			fakeClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Virtual_Guest_Service_activatePublicPort.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("activates public port for virtual guest instance", func() {
			activated, err := virtualGuestService.ActivatePublicPort(virtualGuest.Id)

			Expect(err).ToNot(HaveOccurred())
			Expect(activated).To(BeTrue())
		})
	})

	Context("#ShutdownPrivatePort", func() {
		BeforeEach(func() {
			virtualGuest.Id = 1234567
			fakeClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Virtual_Guest_Service_shutdownPrivatePort.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("shutdown private port for virtual guest instance", func() {
			shutdowned, err := virtualGuestService.ShutdownPrivatePort(virtualGuest.Id)

			Expect(err).ToNot(HaveOccurred())
			Expect(shutdowned).To(BeTrue())
		})
	})

	Context("#ShutdownPublicPort", func() {
		BeforeEach(func() {
			virtualGuest.Id = 1234567
			fakeClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Virtual_Guest_Service_shutdownPublicPort.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("shuts down public port for virtual guest instance", func() {
			shutdowned, err := virtualGuestService.ShutdownPublicPort(virtualGuest.Id)

			Expect(err).ToNot(HaveOccurred())
			Expect(shutdowned).To(BeTrue())
		})
	})

	Context("#GetAllowedHost", func() {
		BeforeEach(func() {
			virtualGuest.Id = 1234567
			fakeClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Virtual_Guest_Service_getAllowedHost.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("gets allowed host for virtual guest", func() {
			allowedHost, err := virtualGuestService.GetAllowedHost(virtualGuest.Id)

			Expect(err).ToNot(HaveOccurred())
			Expect(allowedHost).NotTo(BeNil())
			Expect(allowedHost.Name).To(Equal("fake-iqn"))
		})
	})

	Context("#GetNetworkVlans", func() {
		BeforeEach(func() {
			virtualGuest.Id = 1234567
			fakeClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Virtual_Guest_Service_getNetworkVlans.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("gets network vlans for virtual guest", func() {
			networkVlans, err := virtualGuestService.GetNetworkVlans(virtualGuest.Id)

			Expect(err).ToNot(HaveOccurred())
			Expect(len(networkVlans)).To(Equal(2))
			Expect(networkVlans[0].AccountId).To(Equal(278444))
			Expect(networkVlans[0].Id).To(Equal(293731))
			Expect(networkVlans[0].ModifyDate).ToNot(BeNil())
			Expect(networkVlans[0].Name).To(Equal("AMS CLMS Pub"))
			Expect(networkVlans[0].NetworkVrfId).To(Equal(0))
			Expect(networkVlans[0].Note).To(Equal(""))
			Expect(networkVlans[0].PrimarySubnetId).To(Equal(517311))
			Expect(networkVlans[0].VlanNumber).To(Equal(809))
		})
	})

	Context("#CheckHostDiskAvailability", func() {
		BeforeEach(func() {
			virtualGuest.Id = 1234567
			fakeClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Virtual_Guest_Service_checkHostDiskAvailability.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("checks for host disk availability", func() {
			available, err := virtualGuestService.CheckHostDiskAvailability(virtualGuest.Id, 10*1024)

			Expect(err).ToNot(HaveOccurred())
			Expect(available).To(BeTrue())
		})
	})

	Context("#CaptureImage", func() {
		BeforeEach(func() {
			virtualGuest.Id = 1234567
			fakeClient.DoRawHttpRequestResponse, err = testhelpers.ReadJsonTestFixtures("services", "SoftLayer_Virtual_Guest_Service_captureImage.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("captures the virtual guest as a container disk image template", func() {
			diskImageTemplate, err := virtualGuestService.CaptureImage(virtualGuest.Id)

			Expect(err).ToNot(HaveOccurred())
			Expect(diskImageTemplate.Description).To(Equal("fake-description"))
			Expect(diskImageTemplate.Name).To(Equal("fake-name"))
			Expect(diskImageTemplate.Summary).To(Equal("fake-summary"))
			Expect(len(diskImageTemplate.Volumes)).To(BeNumerically(">=", 1))
			Expect(diskImageTemplate.Volumes[0].Name).To(Equal("fake-volume-name"))
			Expect(len(diskImageTemplate.Volumes[0].Partitions)).To(BeNumerically(">=", 1))
			Expect(diskImageTemplate.Volumes[0].Partitions[0].Name).To(Equal("fake-partition-name"))
		})
	})
})
