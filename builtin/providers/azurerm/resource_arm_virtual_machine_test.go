package azurerm

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/arm/compute"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAzureRMVirtualMachine_basicLinuxMachine(t *testing.T) {
	var vm compute.VirtualMachine
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMVirtualMachine_basicLinuxMachine, ri, ri, ri, ri, ri, ri, ri)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMVirtualMachineDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualMachineExists("azurerm_virtual_machine.test", &vm),
				),
			},
		},
	})
}

func TestAccAzureRMVirtualMachine_basicLinuxMachine_disappears(t *testing.T) {
	var vm compute.VirtualMachine
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMVirtualMachine_basicLinuxMachine, ri, ri, ri, ri, ri, ri, ri)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMVirtualMachineDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualMachineExists("azurerm_virtual_machine.test", &vm),
					testCheckAzureRMVirtualMachineDisappears("azurerm_virtual_machine.test", &vm),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccAzureRMVirtualMachine_withDataDisk(t *testing.T) {
	var vm compute.VirtualMachine

	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMVirtualMachine_withDataDisk, ri, ri, ri, ri, ri, ri, ri)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMVirtualMachineDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualMachineExists("azurerm_virtual_machine.test", &vm),
				),
			},
		},
	})
}

func TestAccAzureRMVirtualMachine_tags(t *testing.T) {
	var vm compute.VirtualMachine

	ri := acctest.RandInt()
	preConfig := fmt.Sprintf(testAccAzureRMVirtualMachine_basicLinuxMachine, ri, ri, ri, ri, ri, ri, ri)
	postConfig := fmt.Sprintf(testAccAzureRMVirtualMachine_basicLinuxMachineUpdated, ri, ri, ri, ri, ri, ri, ri)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMVirtualMachineDestroy,
		Steps: []resource.TestStep{
			{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualMachineExists("azurerm_virtual_machine.test", &vm),
					resource.TestCheckResourceAttr(
						"azurerm_virtual_machine.test", "tags.%", "2"),
					resource.TestCheckResourceAttr(
						"azurerm_virtual_machine.test", "tags.environment", "Production"),
					resource.TestCheckResourceAttr(
						"azurerm_virtual_machine.test", "tags.cost-center", "Ops"),
				),
			},

			{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualMachineExists("azurerm_virtual_machine.test", &vm),
					resource.TestCheckResourceAttr(
						"azurerm_virtual_machine.test", "tags.%", "1"),
					resource.TestCheckResourceAttr(
						"azurerm_virtual_machine.test", "tags.environment", "Production"),
				),
			},
		},
	})
}

//This is a regression test around https://github.com/hashicorp/terraform/issues/6517
//Because we use CreateOrUpdate, we were sending an empty password on update requests
func TestAccAzureRMVirtualMachine_updateMachineSize(t *testing.T) {
	var vm compute.VirtualMachine

	ri := acctest.RandInt()
	preConfig := fmt.Sprintf(testAccAzureRMVirtualMachine_basicLinuxMachine, ri, ri, ri, ri, ri, ri, ri)
	postConfig := fmt.Sprintf(testAccAzureRMVirtualMachine_updatedLinuxMachine, ri, ri, ri, ri, ri, ri, ri)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMVirtualMachineDestroy,
		Steps: []resource.TestStep{
			{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualMachineExists("azurerm_virtual_machine.test", &vm),
					resource.TestCheckResourceAttr(
						"azurerm_virtual_machine.test", "vm_size", "Standard_A0"),
				),
			},
			{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualMachineExists("azurerm_virtual_machine.test", &vm),
					resource.TestCheckResourceAttr(
						"azurerm_virtual_machine.test", "vm_size", "Standard_A1"),
				),
			},
		},
	})
}

func TestAccAzureRMVirtualMachine_basicWindowsMachine(t *testing.T) {
	var vm compute.VirtualMachine
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMVirtualMachine_basicWindowsMachine, ri, ri, ri, ri, ri, ri)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMVirtualMachineDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualMachineExists("azurerm_virtual_machine.test", &vm),
				),
			},
		},
	})
}

func TestAccAzureRMVirtualMachine_windowsUnattendedConfig(t *testing.T) {
	var vm compute.VirtualMachine
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMVirtualMachine_windowsUnattendedConfig, ri, ri, ri, ri, ri, ri)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMVirtualMachineDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualMachineExists("azurerm_virtual_machine.test", &vm),
				),
			},
		},
	})
}

func TestAccAzureRMVirtualMachine_diagnosticsProfile(t *testing.T) {
	var vm compute.VirtualMachine
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMVirtualMachine_diagnosticsProfile, ri, ri, ri, ri, ri, ri)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMVirtualMachineDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualMachineExists("azurerm_virtual_machine.test", &vm),
				),
			},
		},
	})
}

func TestAccAzureRMVirtualMachine_winRMConfig(t *testing.T) {
	var vm compute.VirtualMachine
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMVirtualMachine_winRMConfig, ri, ri, ri, ri, ri, ri)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMVirtualMachineDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualMachineExists("azurerm_virtual_machine.test", &vm),
				),
			},
		},
	})
}

func TestAccAzureRMVirtualMachine_deleteVHDOptOut(t *testing.T) {
	var vm compute.VirtualMachine
	ri := acctest.RandInt()
	preConfig := fmt.Sprintf(testAccAzureRMVirtualMachine_withDataDisk, ri, ri, ri, ri, ri, ri, ri)
	postConfig := fmt.Sprintf(testAccAzureRMVirtualMachine_basicLinuxMachineDeleteVM, ri, ri, ri, ri, ri)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMVirtualMachineDestroy,
		Steps: []resource.TestStep{
			{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualMachineExists("azurerm_virtual_machine.test", &vm),
				),
			},
			{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualMachineVHDExistance("myosdisk1.vhd", true),
					testCheckAzureRMVirtualMachineVHDExistance("mydatadisk1.vhd", true),
				),
			},
		},
	})
}

func TestAccAzureRMVirtualMachine_deleteVHDOptIn(t *testing.T) {
	var vm compute.VirtualMachine
	ri := acctest.RandInt()
	preConfig := fmt.Sprintf(testAccAzureRMVirtualMachine_basicLinuxMachineDestroyDisksBefore, ri, ri, ri, ri, ri, ri, ri, ri)
	postConfig := fmt.Sprintf(testAccAzureRMVirtualMachine_basicLinuxMachineDestroyDisksAfter, ri, ri, ri, ri, ri, ri)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMVirtualMachineDestroy,
		Steps: []resource.TestStep{
			{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualMachineExists("azurerm_virtual_machine.test", &vm),
				),
			},
			{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualMachineVHDExistance("myosdisk1.vhd", false),
					testCheckAzureRMVirtualMachineVHDExistance("mydatadisk1.vhd", false),
				),
			},
		},
	})
}

func TestAccAzureRMVirtualMachine_ChangeComputerName(t *testing.T) {
	var afterCreate, afterUpdate compute.VirtualMachine

	ri := acctest.RandInt()
	preConfig := fmt.Sprintf(testAccAzureRMVirtualMachine_machineNameBeforeUpdate, ri, ri, ri, ri, ri, ri, ri)
	postConfig := fmt.Sprintf(testAccAzureRMVirtualMachine_updateMachineName, ri, ri, ri, ri, ri, ri, ri)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMVirtualMachineDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualMachineExists("azurerm_virtual_machine.test", &afterCreate),
				),
			},

			resource.TestStep{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualMachineExists("azurerm_virtual_machine.test", &afterUpdate),
					testAccCheckVirtualMachineRecreated(
						t, &afterCreate, &afterUpdate),
				),
			},
		},
	})
}

func TestAccAzureRMVirtualMachine_ChangeAvailbilitySet(t *testing.T) {
	var afterCreate, afterUpdate compute.VirtualMachine

	ri := acctest.RandInt()
	preConfig := fmt.Sprintf(testAccAzureRMVirtualMachine_withAvailabilitySet, ri, ri, ri, ri, ri, ri, ri, ri)
	postConfig := fmt.Sprintf(testAccAzureRMVirtualMachine_updateAvailabilitySet, ri, ri, ri, ri, ri, ri, ri, ri)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMVirtualMachineDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualMachineExists("azurerm_virtual_machine.test", &afterCreate),
				),
			},

			resource.TestStep{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualMachineExists("azurerm_virtual_machine.test", &afterUpdate),
					testAccCheckVirtualMachineRecreated(
						t, &afterCreate, &afterUpdate),
				),
			},
		},
	})
}

func TestAccAzureRMVirtualMachine_changeStorageImageReference(t *testing.T) {
	var afterCreate, afterUpdate compute.VirtualMachine

	ri := acctest.RandInt()
	preConfig := fmt.Sprintf(testAccAzureRMVirtualMachine_basicLinuxMachineStorageImageBefore, ri, ri, ri, ri, ri, ri, ri)
	postConfig := fmt.Sprintf(testAccAzureRMVirtualMachine_basicLinuxMachineStorageImageAfter, ri, ri, ri, ri, ri, ri, ri)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMVirtualMachineDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualMachineExists("azurerm_virtual_machine.test", &afterCreate),
				),
			},

			resource.TestStep{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualMachineExists("azurerm_virtual_machine.test", &afterUpdate),
					testAccCheckVirtualMachineRecreated(
						t, &afterCreate, &afterUpdate),
				),
			},
		},
	})
}

func TestAccAzureRMVirtualMachine_changeOSDiskVhdUri(t *testing.T) {
	var afterCreate, afterUpdate compute.VirtualMachine

	ri := acctest.RandInt()
	preConfig := fmt.Sprintf(testAccAzureRMVirtualMachine_basicLinuxMachine, ri, ri, ri, ri, ri, ri, ri)
	postConfig := fmt.Sprintf(testAccAzureRMVirtualMachine_basicLinuxMachineWithOSDiskVhdUriChanged, ri, ri, ri, ri, ri, ri, ri)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMVirtualMachineDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualMachineExists("azurerm_virtual_machine.test", &afterCreate),
				),
			},

			resource.TestStep{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualMachineExists("azurerm_virtual_machine.test", &afterUpdate),
					testAccCheckVirtualMachineRecreated(
						t, &afterCreate, &afterUpdate),
				),
			},
		},
	})
}

func TestAccAzureRMVirtualMachine_plan(t *testing.T) {
	var vm compute.VirtualMachine
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMVirtualMachine_plan, ri, ri, ri, ri, ri, ri, ri)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMVirtualMachineDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualMachineExists("azurerm_virtual_machine.test", &vm),
				),
			},
		},
	})
}

func TestAccAzureRMVirtualMachine_changeSSHKey(t *testing.T) {
	var vm compute.VirtualMachine
	ri := strings.ToLower(acctest.RandString(10))
	preConfig := fmt.Sprintf(testAccAzureRMVirtualMachine_linuxMachineWithSSH, ri, ri, ri, ri, ri, ri, ri)
	postConfig := fmt.Sprintf(testAccAzureRMVirtualMachine_linuxMachineWithSSHRemoved, ri, ri, ri, ri, ri, ri, ri)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMVirtualMachineDestroy,
		Steps: []resource.TestStep{
			{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualMachineExists("azurerm_virtual_machine.test", &vm),
				),
			},
			{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualMachineExists("azurerm_virtual_machine.test", &vm),
				),
			},
		},
	})
}

func testCheckAzureRMVirtualMachineExists(name string, vm *compute.VirtualMachine) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		vmName := rs.Primary.Attributes["name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for virtual machine: %s", vmName)
		}

		conn := testAccProvider.Meta().(*ArmClient).vmClient

		resp, err := conn.Get(resourceGroup, vmName, "")
		if err != nil {
			return fmt.Errorf("Bad: Get on vmClient: %s", err)
		}

		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("Bad: VirtualMachine %q (resource group: %q) does not exist", vmName, resourceGroup)
		}

		*vm = resp

		return nil
	}
}

func testAccCheckVirtualMachineRecreated(t *testing.T,
	before, after *compute.VirtualMachine) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if before.ID == after.ID {
			t.Fatalf("Expected change of Virtual Machine IDs, but both were %v", before.ID)
		}
		return nil
	}
}

func testCheckAzureRMVirtualMachineDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*ArmClient).vmClient

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_virtual_machine" {
			continue
		}

		name := rs.Primary.Attributes["name"]
		resourceGroup := rs.Primary.Attributes["resource_group_name"]

		resp, err := conn.Get(resourceGroup, name, "")

		if err != nil {
			return nil
		}

		if resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("Virtual Machine still exists:\n%#v", resp.VirtualMachineProperties)
		}
	}

	return nil
}

func testCheckAzureRMVirtualMachineVHDExistance(name string, shouldExist bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "azurerm_storage_container" {
				continue
			}

			// fetch storage account and container name
			resourceGroup := rs.Primary.Attributes["resource_group_name"]
			storageAccountName := rs.Primary.Attributes["storage_account_name"]
			containerName := rs.Primary.Attributes["name"]
			storageClient, _, err := testAccProvider.Meta().(*ArmClient).getBlobStorageClientForStorageAccount(resourceGroup, storageAccountName)
			if err != nil {
				return fmt.Errorf("Error creating Blob storage client: %s", err)
			}

			exists, err := storageClient.BlobExists(containerName, name)
			if err != nil {
				return fmt.Errorf("Error checking if Disk VHD Blob exists: %s", err)
			}

			if exists && !shouldExist {
				return fmt.Errorf("Disk VHD Blob still exists")
			} else if !exists && shouldExist {
				return fmt.Errorf("Disk VHD Blob should exist")
			}
		}

		return nil
	}
}

func testCheckAzureRMVirtualMachineDisappears(name string, vm *compute.VirtualMachine) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		vmName := rs.Primary.Attributes["name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for virtual machine: %s", vmName)
		}

		conn := testAccProvider.Meta().(*ArmClient).vmClient

		_, err := conn.Delete(resourceGroup, vmName, make(chan struct{}))
		if err != nil {
			return fmt.Errorf("Bad: Delete on vmClient: %s", err)
		}

		return nil
	}
}

func TestAccAzureRMVirtualMachine_windowsLicenseType(t *testing.T) {
	var vm compute.VirtualMachine
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMVirtualMachine_windowsLicenseType, ri, ri, ri, ri, ri, ri)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMVirtualMachineDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualMachineExists("azurerm_virtual_machine.test", &vm),
				),
			},
		},
	})
}

func TestAccAzureRMVirtualMachine_primaryNetworkInterfaceId(t *testing.T) {
	var vm compute.VirtualMachine
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMVirtualMachine_primaryNetworkInterfaceId, ri, ri, ri, ri, ri, ri, ri)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMVirtualMachineDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualMachineExists("azurerm_virtual_machine.test", &vm),
				),
			},
		},
	})
}

var testAccAzureRMVirtualMachine_basicLinuxMachine = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
}

resource "azurerm_virtual_network" "test" {
    name = "acctvn-%d"
    address_space = ["10.0.0.0/16"]
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_subnet" "test" {
    name = "acctsub-%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    virtual_network_name = "${azurerm_virtual_network.test.name}"
    address_prefix = "10.0.2.0/24"
}

resource "azurerm_network_interface" "test" {
    name = "acctni-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"

    ip_configuration {
    	name = "testconfiguration1"
    	subnet_id = "${azurerm_subnet.test.id}"
    	private_ip_address_allocation = "dynamic"
    }
}

resource "azurerm_storage_account" "test" {
    name = "accsa%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    location = "westus"
    account_type = "Standard_LRS"

    tags {
        environment = "staging"
    }
}

resource "azurerm_storage_container" "test" {
    name = "vhds"
    resource_group_name = "${azurerm_resource_group.test.name}"
    storage_account_name = "${azurerm_storage_account.test.name}"
    container_access_type = "private"
}

resource "azurerm_virtual_machine" "test" {
    name = "acctvm-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    network_interface_ids = ["${azurerm_network_interface.test.id}"]
    vm_size = "Standard_A0"

    storage_image_reference {
	publisher = "Canonical"
	offer = "UbuntuServer"
	sku = "14.04.2-LTS"
	version = "latest"
    }

    storage_os_disk {
        name = "myosdisk1"
        vhd_uri = "${azurerm_storage_account.test.primary_blob_endpoint}${azurerm_storage_container.test.name}/myosdisk1.vhd"
        caching = "ReadWrite"
        create_option = "FromImage"
        disk_size_gb = "45"
    }

    os_profile {
	computer_name = "hostname%d"
	admin_username = "testadmin"
	admin_password = "Password1234!"
    }

    os_profile_linux_config {
	disable_password_authentication = false
    }

    tags {
    	environment = "Production"
    	cost-center = "Ops"
    }
}
`

var testAccAzureRMVirtualMachine_machineNameBeforeUpdate = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
}

resource "azurerm_virtual_network" "test" {
    name = "acctvn-%d"
    address_space = ["10.0.0.0/16"]
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_subnet" "test" {
    name = "acctsub-%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    virtual_network_name = "${azurerm_virtual_network.test.name}"
    address_prefix = "10.0.2.0/24"
}

resource "azurerm_network_interface" "test" {
    name = "acctni-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"

    ip_configuration {
    	name = "testconfiguration1"
    	subnet_id = "${azurerm_subnet.test.id}"
    	private_ip_address_allocation = "dynamic"
    }
}

resource "azurerm_storage_account" "test" {
    name = "accsa%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    location = "westus"
    account_type = "Standard_LRS"

    tags {
        environment = "staging"
    }
}

resource "azurerm_storage_container" "test" {
    name = "vhds"
    resource_group_name = "${azurerm_resource_group.test.name}"
    storage_account_name = "${azurerm_storage_account.test.name}"
    container_access_type = "private"
}

resource "azurerm_virtual_machine" "test" {
    name = "acctvm-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    network_interface_ids = ["${azurerm_network_interface.test.id}"]
    vm_size = "Standard_A0"
    delete_os_disk_on_termination = true

    storage_image_reference {
	publisher = "Canonical"
	offer = "UbuntuServer"
	sku = "14.04.2-LTS"
	version = "latest"
    }

    storage_os_disk {
        name = "myosdisk1"
        vhd_uri = "${azurerm_storage_account.test.primary_blob_endpoint}${azurerm_storage_container.test.name}/myosdisk1.vhd"
        caching = "ReadWrite"
        create_option = "FromImage"
    }

    os_profile {
	computer_name = "hostname%d"
	admin_username = "testadmin"
	admin_password = "Password1234!"
    }

    os_profile_linux_config {
	disable_password_authentication = false
    }

    tags {
    	environment = "Production"
    	cost-center = "Ops"
    }
}
`

var testAccAzureRMVirtualMachine_basicLinuxMachineDestroyDisksBefore = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
}

resource "azurerm_resource_group" "test-sa" {
    name = "acctestRG-sa-%d"
    location = "West US"
}

resource "azurerm_virtual_network" "test" {
    name = "acctvn-%d"
    address_space = ["10.0.0.0/16"]
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_subnet" "test" {
    name = "acctsub-%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    virtual_network_name = "${azurerm_virtual_network.test.name}"
    address_prefix = "10.0.2.0/24"
}

resource "azurerm_network_interface" "test" {
    name = "acctni-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"

    ip_configuration {
    	name = "testconfiguration1"
    	subnet_id = "${azurerm_subnet.test.id}"
    	private_ip_address_allocation = "dynamic"
    }
}

resource "azurerm_storage_account" "test" {
    name = "accsa%d"
    resource_group_name = "${azurerm_resource_group.test-sa.name}"
    location = "westus"
    account_type = "Standard_LRS"

    tags {
        environment = "staging"
    }
}

resource "azurerm_storage_container" "test" {
    name = "vhds"
    resource_group_name = "${azurerm_resource_group.test-sa.name}"
    storage_account_name = "${azurerm_storage_account.test.name}"
    container_access_type = "private"
}

resource "azurerm_virtual_machine" "test" {
    name = "acctvm-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    network_interface_ids = ["${azurerm_network_interface.test.id}"]
    vm_size = "Standard_A0"

    storage_image_reference {
	publisher = "Canonical"
	offer = "UbuntuServer"
	sku = "14.04.2-LTS"
	version = "latest"
    }

    storage_os_disk {
        name = "myosdisk1"
        vhd_uri = "${azurerm_storage_account.test.primary_blob_endpoint}${azurerm_storage_container.test.name}/myosdisk1.vhd"
        caching = "ReadWrite"
        create_option = "FromImage"
    }

    delete_os_disk_on_termination = true

    storage_data_disk {
        name          = "mydatadisk1"
        vhd_uri       = "${azurerm_storage_account.test.primary_blob_endpoint}${azurerm_storage_container.test.name}/mydatadisk1.vhd"
    	disk_size_gb  = "1023"
    	create_option = "Empty"
    	lun           = 0
    }

    delete_data_disks_on_termination = true

    os_profile {
	computer_name = "hostname%d"
	admin_username = "testadmin"
	admin_password = "Password1234!"
    }

    os_profile_linux_config {
	disable_password_authentication = false
    }

    tags {
    	environment = "Production"
    	cost-center = "Ops"
    }
}
`

var testAccAzureRMVirtualMachine_basicLinuxMachineDestroyDisksAfter = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
}

resource "azurerm_resource_group" "test-sa" {
    name = "acctestRG-sa-%d"
    location = "West US"
}

resource "azurerm_virtual_network" "test" {
    name = "acctvn-%d"
    address_space = ["10.0.0.0/16"]
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_subnet" "test" {
    name = "acctsub-%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    virtual_network_name = "${azurerm_virtual_network.test.name}"
    address_prefix = "10.0.2.0/24"
}

resource "azurerm_network_interface" "test" {
    name = "acctni-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"

    ip_configuration {
    	name = "testconfiguration1"
    	subnet_id = "${azurerm_subnet.test.id}"
    	private_ip_address_allocation = "dynamic"
    }
}

resource "azurerm_storage_account" "test" {
    name = "accsa%d"
    resource_group_name = "${azurerm_resource_group.test-sa.name}"
    location = "westus"
    account_type = "Standard_LRS"

    tags {
        environment = "staging"
    }
}

resource "azurerm_storage_container" "test" {
    name = "vhds"
    resource_group_name = "${azurerm_resource_group.test-sa.name}"
    storage_account_name = "${azurerm_storage_account.test.name}"
    container_access_type = "private"
}
`

var testAccAzureRMVirtualMachine_basicLinuxMachineDeleteVM = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
}

resource "azurerm_virtual_network" "test" {
    name = "acctvn-%d"
    address_space = ["10.0.0.0/16"]
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_subnet" "test" {
    name = "acctsub-%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    virtual_network_name = "${azurerm_virtual_network.test.name}"
    address_prefix = "10.0.2.0/24"
}

resource "azurerm_network_interface" "test" {
    name = "acctni-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"

    ip_configuration {
    	name = "testconfiguration1"
    	subnet_id = "${azurerm_subnet.test.id}"
    	private_ip_address_allocation = "dynamic"
    }
}

resource "azurerm_storage_account" "test" {
    name = "accsa%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    location = "westus"
    account_type = "Standard_LRS"

    tags {
        environment = "staging"
    }
}

resource "azurerm_storage_container" "test" {
    name = "vhds"
    resource_group_name = "${azurerm_resource_group.test.name}"
    storage_account_name = "${azurerm_storage_account.test.name}"
    container_access_type = "private"
}
`

var testAccAzureRMVirtualMachine_withDataDisk = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
}

resource "azurerm_virtual_network" "test" {
    name = "acctvn-%d"
    address_space = ["10.0.0.0/16"]
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_subnet" "test" {
    name = "acctsub-%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    virtual_network_name = "${azurerm_virtual_network.test.name}"
    address_prefix = "10.0.2.0/24"
}

resource "azurerm_network_interface" "test" {
    name = "acctni-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"

    ip_configuration {
    	name = "testconfiguration1"
    	subnet_id = "${azurerm_subnet.test.id}"
    	private_ip_address_allocation = "dynamic"
    }
}

resource "azurerm_storage_account" "test" {
    name = "accsa%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    location = "westus"
    account_type = "Standard_LRS"

    tags {
        environment = "staging"
    }
}

resource "azurerm_storage_container" "test" {
    name = "vhds"
    resource_group_name = "${azurerm_resource_group.test.name}"
    storage_account_name = "${azurerm_storage_account.test.name}"
    container_access_type = "private"
}

resource "azurerm_virtual_machine" "test" {
    name = "acctvm-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    network_interface_ids = ["${azurerm_network_interface.test.id}"]
    vm_size = "Standard_A0"

    storage_image_reference {
	publisher = "Canonical"
	offer = "UbuntuServer"
	sku = "14.04.2-LTS"
	version = "latest"
    }

    storage_os_disk {
        name = "myosdisk1"
        vhd_uri = "${azurerm_storage_account.test.primary_blob_endpoint}${azurerm_storage_container.test.name}/myosdisk1.vhd"
        caching = "ReadWrite"
        create_option = "FromImage"
    }

    storage_data_disk {
        name          = "mydatadisk1"
        vhd_uri       = "${azurerm_storage_account.test.primary_blob_endpoint}${azurerm_storage_container.test.name}/mydatadisk1.vhd"
    	disk_size_gb  = "1023"
    	create_option = "Empty"
        caching       = "ReadWrite"
    	lun           = 0
    }

    os_profile {
	computer_name = "hostname%d"
	admin_username = "testadmin"
	admin_password = "Password1234!"
    }

    os_profile_linux_config {
	disable_password_authentication = false
    }

    tags {
    	environment = "Production"
    	cost-center = "Ops"
    }
}
`

var testAccAzureRMVirtualMachine_basicLinuxMachineUpdated = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
}

resource "azurerm_virtual_network" "test" {
    name = "acctvn-%d"
    address_space = ["10.0.0.0/16"]
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_subnet" "test" {
    name = "acctsub-%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    virtual_network_name = "${azurerm_virtual_network.test.name}"
    address_prefix = "10.0.2.0/24"
}

resource "azurerm_network_interface" "test" {
    name = "acctni-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"

    ip_configuration {
    	name = "testconfiguration1"
    	subnet_id = "${azurerm_subnet.test.id}"
    	private_ip_address_allocation = "dynamic"
    }
}

resource "azurerm_storage_account" "test" {
    name = "accsa%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    location = "westus"
    account_type = "Standard_LRS"

    tags {
        environment = "staging"
    }
}

resource "azurerm_storage_container" "test" {
    name = "vhds"
    resource_group_name = "${azurerm_resource_group.test.name}"
    storage_account_name = "${azurerm_storage_account.test.name}"
    container_access_type = "private"
}

resource "azurerm_virtual_machine" "test" {
    name = "acctvm-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    network_interface_ids = ["${azurerm_network_interface.test.id}"]
    vm_size = "Standard_A0"

    storage_image_reference {
	publisher = "Canonical"
	offer = "UbuntuServer"
	sku = "14.04.2-LTS"
	version = "latest"
    }

    storage_os_disk {
        name = "myosdisk1"
        vhd_uri = "${azurerm_storage_account.test.primary_blob_endpoint}${azurerm_storage_container.test.name}/myosdisk1.vhd"
        caching = "ReadWrite"
        create_option = "FromImage"
    }

    os_profile {
	computer_name = "hostname%d"
	admin_username = "testadmin"
	admin_password = "Password1234!"
    }

    os_profile_linux_config {
	disable_password_authentication = false
    }

    tags {
    	environment = "Production"
    }
}
`

var testAccAzureRMVirtualMachine_updatedLinuxMachine = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
}

resource "azurerm_virtual_network" "test" {
    name = "acctvn-%d"
    address_space = ["10.0.0.0/16"]
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_subnet" "test" {
    name = "acctsub-%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    virtual_network_name = "${azurerm_virtual_network.test.name}"
    address_prefix = "10.0.2.0/24"
}

resource "azurerm_network_interface" "test" {
    name = "acctni-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"

    ip_configuration {
    	name = "testconfiguration1"
    	subnet_id = "${azurerm_subnet.test.id}"
    	private_ip_address_allocation = "dynamic"
    }
}

resource "azurerm_storage_account" "test" {
    name = "accsa%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    location = "westus"
    account_type = "Standard_LRS"

    tags {
        environment = "staging"
    }
}

resource "azurerm_storage_container" "test" {
    name = "vhds"
    resource_group_name = "${azurerm_resource_group.test.name}"
    storage_account_name = "${azurerm_storage_account.test.name}"
    container_access_type = "private"
}

resource "azurerm_virtual_machine" "test" {
    name = "acctvm-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    network_interface_ids = ["${azurerm_network_interface.test.id}"]
    vm_size = "Standard_A1"

    storage_image_reference {
	publisher = "Canonical"
	offer = "UbuntuServer"
	sku = "14.04.2-LTS"
	version = "latest"
    }

    storage_os_disk {
        name = "myosdisk1"
        vhd_uri = "${azurerm_storage_account.test.primary_blob_endpoint}${azurerm_storage_container.test.name}/myosdisk1.vhd"
        caching = "ReadWrite"
        create_option = "FromImage"
    }

    os_profile {
	computer_name = "hostname%d"
	admin_username = "testadmin"
	admin_password = "Password1234!"
    }

    os_profile_linux_config {
	disable_password_authentication = false
    }
}
`

var testAccAzureRMVirtualMachine_basicWindowsMachine = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
}

resource "azurerm_virtual_network" "test" {
    name = "acctvn-%d"
    address_space = ["10.0.0.0/16"]
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_subnet" "test" {
    name = "acctsub-%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    virtual_network_name = "${azurerm_virtual_network.test.name}"
    address_prefix = "10.0.2.0/24"
}

resource "azurerm_network_interface" "test" {
    name = "acctni-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"

    ip_configuration {
    	name = "testconfiguration1"
    	subnet_id = "${azurerm_subnet.test.id}"
    	private_ip_address_allocation = "dynamic"
    }
}

resource "azurerm_storage_account" "test" {
    name = "accsa%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    location = "westus"
    account_type = "Standard_LRS"

    tags {
        environment = "staging"
    }
}

resource "azurerm_storage_container" "test" {
    name = "vhds"
    resource_group_name = "${azurerm_resource_group.test.name}"
    storage_account_name = "${azurerm_storage_account.test.name}"
    container_access_type = "private"
}

resource "azurerm_virtual_machine" "test" {
    name = "acctvm-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    network_interface_ids = ["${azurerm_network_interface.test.id}"]
    vm_size = "Standard_A0"

    storage_image_reference {
	publisher = "MicrosoftWindowsServer"
	offer = "WindowsServer"
	sku = "2012-Datacenter"
	version = "latest"
    }

    storage_os_disk {
        name = "myosdisk1"
        vhd_uri = "${azurerm_storage_account.test.primary_blob_endpoint}${azurerm_storage_container.test.name}/myosdisk1.vhd"
        caching = "ReadWrite"
        create_option = "FromImage"
    }

    os_profile {
	computer_name = "winhost01"
	admin_username = "testadmin"
	admin_password = "Password1234!"
    }

    os_profile_windows_config {
	enable_automatic_upgrades = false
	provision_vm_agent = true
    }
}
`

var testAccAzureRMVirtualMachine_windowsUnattendedConfig = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
}

resource "azurerm_virtual_network" "test" {
    name = "acctvn-%d"
    address_space = ["10.0.0.0/16"]
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_subnet" "test" {
    name = "acctsub-%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    virtual_network_name = "${azurerm_virtual_network.test.name}"
    address_prefix = "10.0.2.0/24"
}

resource "azurerm_network_interface" "test" {
    name = "acctni-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"

    ip_configuration {
    	name = "testconfiguration1"
    	subnet_id = "${azurerm_subnet.test.id}"
    	private_ip_address_allocation = "dynamic"
    }
}

resource "azurerm_storage_account" "test" {
    name = "accsa%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    location = "westus"
    account_type = "Standard_LRS"

    tags {
        environment = "staging"
    }
}

resource "azurerm_storage_container" "test" {
    name = "vhds"
    resource_group_name = "${azurerm_resource_group.test.name}"
    storage_account_name = "${azurerm_storage_account.test.name}"
    container_access_type = "private"
}

resource "azurerm_virtual_machine" "test" {
    name = "acctvm-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    network_interface_ids = ["${azurerm_network_interface.test.id}"]
    vm_size = "Standard_A0"

    storage_image_reference {
	publisher = "MicrosoftWindowsServer"
	offer = "WindowsServer"
	sku = "2012-Datacenter"
	version = "latest"
    }

    storage_os_disk {
        name = "myosdisk1"
        vhd_uri = "${azurerm_storage_account.test.primary_blob_endpoint}${azurerm_storage_container.test.name}/myosdisk1.vhd"
        caching = "ReadWrite"
        create_option = "FromImage"
    }

    os_profile {
	computer_name = "winhost01"
	admin_username = "testadmin"
	admin_password = "Password1234!"
    }

    os_profile_windows_config {
        provision_vm_agent = true
        additional_unattend_config {
            pass = "oobeSystem"
            component = "Microsoft-Windows-Shell-Setup"
            setting_name = "FirstLogonCommands"
            content = "<FirstLogonCommands><SynchronousCommand><CommandLine>shutdown /r /t 0 /c \"initial reboot\"</CommandLine><Description>reboot</Description><Order>1</Order></SynchronousCommand></FirstLogonCommands>"
        }
    }

}
`

var testAccAzureRMVirtualMachine_diagnosticsProfile = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
}

resource "azurerm_virtual_network" "test" {
    name = "acctvn-%d"
    address_space = ["10.0.0.0/16"]
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_subnet" "test" {
    name = "acctsub-%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    virtual_network_name = "${azurerm_virtual_network.test.name}"
    address_prefix = "10.0.2.0/24"
}

resource "azurerm_network_interface" "test" {
    name = "acctni-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"

    ip_configuration {
        name = "testconfiguration1"
        subnet_id = "${azurerm_subnet.test.id}"
        private_ip_address_allocation = "dynamic"
    }
}

resource "azurerm_storage_account" "test" {
    name = "accsa%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    location = "westus"
    account_type = "Standard_LRS"

    tags {
        environment = "staging"
    }
}

resource "azurerm_storage_container" "test" {
    name = "vhds"
    resource_group_name = "${azurerm_resource_group.test.name}"
    storage_account_name = "${azurerm_storage_account.test.name}"
    container_access_type = "private"
}

resource "azurerm_virtual_machine" "test" {
    name = "acctvm-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    network_interface_ids = ["${azurerm_network_interface.test.id}"]
    vm_size = "Standard_A0"

    storage_image_reference {
	publisher = "MicrosoftWindowsServer"
	offer = "WindowsServer"
	sku = "2012-Datacenter"
	version = "latest"
    }

    storage_os_disk {
        name = "myosdisk1"
        vhd_uri = "${azurerm_storage_account.test.primary_blob_endpoint}${azurerm_storage_container.test.name}/myosdisk1.vhd"
        caching = "ReadWrite"
        create_option = "FromImage"
    }

    os_profile {
	computer_name = "winhost01"
	admin_username = "testadmin"
	admin_password = "Password1234!"
    }

    boot_diagnostics {
        enabled = true
        storage_uri = "${azurerm_storage_account.test.primary_blob_endpoint}"
    }

    os_profile_windows_config {
        winrm {
	  protocol = "http"
        }
    }
}

`

var testAccAzureRMVirtualMachine_winRMConfig = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
}

resource "azurerm_virtual_network" "test" {
    name = "acctvn-%d"
    address_space = ["10.0.0.0/16"]
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_subnet" "test" {
    name = "acctsub-%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    virtual_network_name = "${azurerm_virtual_network.test.name}"
    address_prefix = "10.0.2.0/24"
}

resource "azurerm_network_interface" "test" {
    name = "acctni-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"

    ip_configuration {
    	name = "testconfiguration1"
    	subnet_id = "${azurerm_subnet.test.id}"
    	private_ip_address_allocation = "dynamic"
    }
}

resource "azurerm_storage_account" "test" {
    name = "accsa%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    location = "westus"
    account_type = "Standard_LRS"

    tags {
        environment = "staging"
    }
}

resource "azurerm_storage_container" "test" {
    name = "vhds"
    resource_group_name = "${azurerm_resource_group.test.name}"
    storage_account_name = "${azurerm_storage_account.test.name}"
    container_access_type = "private"
}

resource "azurerm_virtual_machine" "test" {
    name = "acctvm-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    network_interface_ids = ["${azurerm_network_interface.test.id}"]
    vm_size = "Standard_A0"

    storage_image_reference {
	publisher = "MicrosoftWindowsServer"
	offer = "WindowsServer"
	sku = "2012-Datacenter"
	version = "latest"
    }

    storage_os_disk {
        name = "myosdisk1"
        vhd_uri = "${azurerm_storage_account.test.primary_blob_endpoint}${azurerm_storage_container.test.name}/myosdisk1.vhd"
        caching = "ReadWrite"
        create_option = "FromImage"
    }

    os_profile {
	computer_name = "winhost01"
	admin_username = "testadmin"
	admin_password = "Password1234!"
    }

    os_profile_windows_config {
        winrm {
	  protocol = "http"
        }
    }
}
`

var testAccAzureRMVirtualMachine_withAvailabilitySet = `
 resource "azurerm_resource_group" "test" {
     name = "acctestRG-%d"
     location = "West US"
 }

 resource "azurerm_virtual_network" "test" {
     name = "acctvn-%d"
     address_space = ["10.0.0.0/16"]
     location = "West US"
     resource_group_name = "${azurerm_resource_group.test.name}"
 }

 resource "azurerm_subnet" "test" {
     name = "acctsub-%d"
     resource_group_name = "${azurerm_resource_group.test.name}"
     virtual_network_name = "${azurerm_virtual_network.test.name}"
     address_prefix = "10.0.2.0/24"
 }

 resource "azurerm_network_interface" "test" {
     name = "acctni-%d"
     location = "West US"
     resource_group_name = "${azurerm_resource_group.test.name}"

     ip_configuration {
     	name = "testconfiguration1"
     	subnet_id = "${azurerm_subnet.test.id}"
     	private_ip_address_allocation = "dynamic"
     }
 }

 resource "azurerm_storage_account" "test" {
     name = "accsa%d"
     resource_group_name = "${azurerm_resource_group.test.name}"
     location = "westus"
     account_type = "Standard_LRS"

     tags {
         environment = "staging"
     }
 }

 resource "azurerm_availability_set" "test" {
    name = "availabilityset%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

 resource "azurerm_storage_container" "test" {
     name = "vhds"
     resource_group_name = "${azurerm_resource_group.test.name}"
     storage_account_name = "${azurerm_storage_account.test.name}"
     container_access_type = "private"
 }

 resource "azurerm_virtual_machine" "test" {
     name = "acctvm-%d"
     location = "West US"
     resource_group_name = "${azurerm_resource_group.test.name}"
     network_interface_ids = ["${azurerm_network_interface.test.id}"]
     vm_size = "Standard_A0"
     availability_set_id = "${azurerm_availability_set.test.id}"
     delete_os_disk_on_termination = true

     storage_image_reference {
 	publisher = "Canonical"
 	offer = "UbuntuServer"
 	sku = "14.04.2-LTS"
 	version = "latest"
     }

     storage_os_disk {
         name = "myosdisk1"
         vhd_uri = "${azurerm_storage_account.test.primary_blob_endpoint}${azurerm_storage_container.test.name}/myosdisk1.vhd"
         caching = "ReadWrite"
         create_option = "FromImage"
     }

     os_profile {
 	computer_name = "hostname%d"
 	admin_username = "testadmin"
 	admin_password = "Password1234!"
     }

     os_profile_linux_config {
 	disable_password_authentication = false
     }
 }
`

var testAccAzureRMVirtualMachine_updateAvailabilitySet = `
 resource "azurerm_resource_group" "test" {
     name = "acctestRG-%d"
     location = "West US"
 }

 resource "azurerm_virtual_network" "test" {
     name = "acctvn-%d"
     address_space = ["10.0.0.0/16"]
     location = "West US"
     resource_group_name = "${azurerm_resource_group.test.name}"
 }

 resource "azurerm_subnet" "test" {
     name = "acctsub-%d"
     resource_group_name = "${azurerm_resource_group.test.name}"
     virtual_network_name = "${azurerm_virtual_network.test.name}"
     address_prefix = "10.0.2.0/24"
 }

 resource "azurerm_network_interface" "test" {
     name = "acctni-%d"
     location = "West US"
     resource_group_name = "${azurerm_resource_group.test.name}"

     ip_configuration {
     	name = "testconfiguration1"
     	subnet_id = "${azurerm_subnet.test.id}"
     	private_ip_address_allocation = "dynamic"
     }
 }

 resource "azurerm_storage_account" "test" {
     name = "accsa%d"
     resource_group_name = "${azurerm_resource_group.test.name}"
     location = "westus"
     account_type = "Standard_LRS"

     tags {
         environment = "staging"
     }
 }

 resource "azurerm_availability_set" "test" {
    name = "updatedAvailabilitySet%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

 resource "azurerm_storage_container" "test" {
     name = "vhds"
     resource_group_name = "${azurerm_resource_group.test.name}"
     storage_account_name = "${azurerm_storage_account.test.name}"
     container_access_type = "private"
 }

 resource "azurerm_virtual_machine" "test" {
     name = "acctvm-%d"
     location = "West US"
     resource_group_name = "${azurerm_resource_group.test.name}"
     network_interface_ids = ["${azurerm_network_interface.test.id}"]
     vm_size = "Standard_A0"
     availability_set_id = "${azurerm_availability_set.test.id}"
     delete_os_disk_on_termination = true

     storage_image_reference {
 	publisher = "Canonical"
 	offer = "UbuntuServer"
 	sku = "14.04.2-LTS"
 	version = "latest"
     }

     storage_os_disk {
         name = "myosdisk1"
         vhd_uri = "${azurerm_storage_account.test.primary_blob_endpoint}${azurerm_storage_container.test.name}/myosdisk1.vhd"
         caching = "ReadWrite"
         create_option = "FromImage"
     }

     os_profile {
 	computer_name = "hostname%d"
 	admin_username = "testadmin"
 	admin_password = "Password1234!"
     }

     os_profile_linux_config {
 	disable_password_authentication = false
     }
 }
`

var testAccAzureRMVirtualMachine_updateMachineName = `
 resource "azurerm_resource_group" "test" {
     name = "acctestRG-%d"
     location = "West US"
 }

 resource "azurerm_virtual_network" "test" {
     name = "acctvn-%d"
     address_space = ["10.0.0.0/16"]
     location = "West US"
     resource_group_name = "${azurerm_resource_group.test.name}"
 }

 resource "azurerm_subnet" "test" {
     name = "acctsub-%d"
     resource_group_name = "${azurerm_resource_group.test.name}"
     virtual_network_name = "${azurerm_virtual_network.test.name}"
     address_prefix = "10.0.2.0/24"
 }

 resource "azurerm_network_interface" "test" {
     name = "acctni-%d"
     location = "West US"
     resource_group_name = "${azurerm_resource_group.test.name}"

     ip_configuration {
     	name = "testconfiguration1"
     	subnet_id = "${azurerm_subnet.test.id}"
     	private_ip_address_allocation = "dynamic"
     }
 }

 resource "azurerm_storage_account" "test" {
     name = "accsa%d"
     resource_group_name = "${azurerm_resource_group.test.name}"
     location = "westus"
     account_type = "Standard_LRS"

     tags {
         environment = "staging"
     }
 }

 resource "azurerm_storage_container" "test" {
     name = "vhds"
     resource_group_name = "${azurerm_resource_group.test.name}"
     storage_account_name = "${azurerm_storage_account.test.name}"
     container_access_type = "private"
 }

 resource "azurerm_virtual_machine" "test" {
     name = "acctvm-%d"
     location = "West US"
     resource_group_name = "${azurerm_resource_group.test.name}"
     network_interface_ids = ["${azurerm_network_interface.test.id}"]
     vm_size = "Standard_A0"
      delete_os_disk_on_termination = true

     storage_image_reference {
 	publisher = "Canonical"
 	offer = "UbuntuServer"
 	sku = "14.04.2-LTS"
 	version = "latest"
     }

     storage_os_disk {
         name = "myosdisk1"
         vhd_uri = "${azurerm_storage_account.test.primary_blob_endpoint}${azurerm_storage_container.test.name}/myosdisk1.vhd"
         caching = "ReadWrite"
         create_option = "FromImage"
     }

     os_profile {
 	computer_name = "newhostname%d"
 	admin_username = "testadmin"
 	admin_password = "Password1234!"
     }

     os_profile_linux_config {
 	disable_password_authentication = false
     }
 }
 `

var testAccAzureRMVirtualMachine_basicLinuxMachineStorageImageBefore = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
}

resource "azurerm_virtual_network" "test" {
    name = "acctvn-%d"
    address_space = ["10.0.0.0/16"]
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_subnet" "test" {
    name = "acctsub-%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    virtual_network_name = "${azurerm_virtual_network.test.name}"
    address_prefix = "10.0.2.0/24"
}

resource "azurerm_network_interface" "test" {
    name = "acctni-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"

    ip_configuration {
    	name = "testconfiguration1"
    	subnet_id = "${azurerm_subnet.test.id}"
    	private_ip_address_allocation = "dynamic"
    }
}

resource "azurerm_storage_account" "test" {
    name = "accsa%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    location = "westus"
    account_type = "Standard_LRS"

    tags {
        environment = "staging"
    }
}

resource "azurerm_storage_container" "test" {
    name = "vhds"
    resource_group_name = "${azurerm_resource_group.test.name}"
    storage_account_name = "${azurerm_storage_account.test.name}"
    container_access_type = "private"
}

resource "azurerm_virtual_machine" "test" {
    name = "acctvm-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    network_interface_ids = ["${azurerm_network_interface.test.id}"]
    vm_size = "Standard_A0"
    delete_os_disk_on_termination = true

    storage_image_reference {
	publisher = "Canonical"
	offer = "UbuntuServer"
	sku = "14.04.2-LTS"
	version = "latest"
    }

    storage_os_disk {
        name = "myosdisk1"
        vhd_uri = "${azurerm_storage_account.test.primary_blob_endpoint}${azurerm_storage_container.test.name}/myosdisk1.vhd"
        caching = "ReadWrite"
        create_option = "FromImage"
        disk_size_gb = "45"
    }

    os_profile {
	computer_name = "hostname%d"
	admin_username = "testadmin"
	admin_password = "Password1234!"
    }

    os_profile_linux_config {
	disable_password_authentication = false
    }

    tags {
    	environment = "Production"
    	cost-center = "Ops"
    }
}
`

var testAccAzureRMVirtualMachine_basicLinuxMachineStorageImageAfter = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
}

resource "azurerm_virtual_network" "test" {
    name = "acctvn-%d"
    address_space = ["10.0.0.0/16"]
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_subnet" "test" {
    name = "acctsub-%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    virtual_network_name = "${azurerm_virtual_network.test.name}"
    address_prefix = "10.0.2.0/24"
}

resource "azurerm_network_interface" "test" {
    name = "acctni-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"

    ip_configuration {
    	name = "testconfiguration1"
    	subnet_id = "${azurerm_subnet.test.id}"
    	private_ip_address_allocation = "dynamic"
    }
}

resource "azurerm_storage_account" "test" {
    name = "accsa%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    location = "westus"
    account_type = "Standard_LRS"

    tags {
        environment = "staging"
    }
}

resource "azurerm_storage_container" "test" {
    name = "vhds"
    resource_group_name = "${azurerm_resource_group.test.name}"
    storage_account_name = "${azurerm_storage_account.test.name}"
    container_access_type = "private"
}

resource "azurerm_virtual_machine" "test" {
    name = "acctvm-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    network_interface_ids = ["${azurerm_network_interface.test.id}"]
    vm_size = "Standard_A0"
    delete_os_disk_on_termination = true

    storage_image_reference {
	publisher = "CoreOS"
	offer = "CoreOS"
	sku = "Stable"
	version = "latest"
    }

    storage_os_disk {
        name = "myosdisk1"
        vhd_uri = "${azurerm_storage_account.test.primary_blob_endpoint}${azurerm_storage_container.test.name}/myosdisk1.vhd"
        caching = "ReadWrite"
        create_option = "FromImage"
        disk_size_gb = "45"
    }

    os_profile {
	computer_name = "hostname%d"
	admin_username = "testadmin"
	admin_password = "Password1234!"
    }

    os_profile_linux_config {
	disable_password_authentication = false
    }

    tags {
    	environment = "Production"
    	cost-center = "Ops"
    }
}
`

var testAccAzureRMVirtualMachine_basicLinuxMachineWithOSDiskVhdUriChanged = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
}

resource "azurerm_virtual_network" "test" {
    name = "acctvn-%d"
    address_space = ["10.0.0.0/16"]
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_subnet" "test" {
    name = "acctsub-%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    virtual_network_name = "${azurerm_virtual_network.test.name}"
    address_prefix = "10.0.2.0/24"
}

resource "azurerm_network_interface" "test" {
    name = "acctni-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"

    ip_configuration {
    	name = "testconfiguration1"
    	subnet_id = "${azurerm_subnet.test.id}"
    	private_ip_address_allocation = "dynamic"
    }
}

resource "azurerm_storage_account" "test" {
    name = "accsa%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    location = "westus"
    account_type = "Standard_LRS"

    tags {
        environment = "staging"
    }
}

resource "azurerm_storage_container" "test" {
    name = "vhds"
    resource_group_name = "${azurerm_resource_group.test.name}"
    storage_account_name = "${azurerm_storage_account.test.name}"
    container_access_type = "private"
}

resource "azurerm_virtual_machine" "test" {
    name = "acctvm-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    network_interface_ids = ["${azurerm_network_interface.test.id}"]
    vm_size = "Standard_A0"

    storage_image_reference {
	publisher = "Canonical"
	offer = "UbuntuServer"
	sku = "14.04.2-LTS"
	version = "latest"
    }

    storage_os_disk {
        name = "myosdisk1"
        vhd_uri = "${azurerm_storage_account.test.primary_blob_endpoint}${azurerm_storage_container.test.name}/myosdiskchanged2.vhd"
        caching = "ReadWrite"
        create_option = "FromImage"
        disk_size_gb = "45"
    }

    os_profile {
	computer_name = "hostname%d"
	admin_username = "testadmin"
	admin_password = "Password1234!"
    }

    os_profile_linux_config {
	disable_password_authentication = false
    }

    tags {
    	environment = "Production"
    	cost-center = "Ops"
    }
}
`

var testAccAzureRMVirtualMachine_windowsLicenseType = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
}

resource "azurerm_virtual_network" "test" {
    name = "acctvn-%d"
    address_space = ["10.0.0.0/16"]
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_subnet" "test" {
    name = "acctsub-%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    virtual_network_name = "${azurerm_virtual_network.test.name}"
    address_prefix = "10.0.2.0/24"
}

resource "azurerm_network_interface" "test" {
    name = "acctni-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"

    ip_configuration {
    	name = "testconfiguration1"
    	subnet_id = "${azurerm_subnet.test.id}"
    	private_ip_address_allocation = "dynamic"
    }
}

resource "azurerm_storage_account" "test" {
    name = "accsa%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    location = "westus"
    account_type = "Standard_LRS"

    tags {
        environment = "staging"
    }
}

resource "azurerm_storage_container" "test" {
    name = "vhds"
    resource_group_name = "${azurerm_resource_group.test.name}"
    storage_account_name = "${azurerm_storage_account.test.name}"
    container_access_type = "private"
}

resource "azurerm_virtual_machine" "test" {
    name = "acctvm-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    network_interface_ids = ["${azurerm_network_interface.test.id}"]
    vm_size = "Standard_A0"
    license_type = "Windows_Server"

    storage_image_reference {
	publisher = "MicrosoftWindowsServer"
	offer = "WindowsServer-HUB"
	sku = "2008-R2-SP1-HUB"
	version = "latest"
    }

    storage_os_disk {
        name = "myosdisk1"
        vhd_uri = "${azurerm_storage_account.test.primary_blob_endpoint}${azurerm_storage_container.test.name}/myosdisk1.vhd"
        caching = "ReadWrite"
        create_option = "FromImage"
    }

    os_profile {
	computer_name = "winhost01"
	admin_username = "testadmin"
	admin_password = "Password1234!"
    }

    os_profile_windows_config {
	enable_automatic_upgrades = false
	provision_vm_agent = true
    }
}
`

var testAccAzureRMVirtualMachine_plan = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
}

resource "azurerm_virtual_network" "test" {
    name = "acctvn-%d"
    address_space = ["10.0.0.0/16"]
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_subnet" "test" {
    name = "acctsub-%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    virtual_network_name = "${azurerm_virtual_network.test.name}"
    address_prefix = "10.0.2.0/24"
}

resource "azurerm_network_interface" "test" {
    name = "acctni-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"

    ip_configuration {
    	name = "testconfiguration1"
    	subnet_id = "${azurerm_subnet.test.id}"
    	private_ip_address_allocation = "dynamic"
    }
}

resource "azurerm_storage_account" "test" {
    name = "accsa%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    location = "westus"
    account_type = "Standard_LRS"

    tags {
        environment = "staging"
    }
}

resource "azurerm_storage_container" "test" {
    name = "vhds"
    resource_group_name = "${azurerm_resource_group.test.name}"
    storage_account_name = "${azurerm_storage_account.test.name}"
    container_access_type = "private"
}

resource "azurerm_virtual_machine" "test" {
    name = "acctvm-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    network_interface_ids = ["${azurerm_network_interface.test.id}"]
    vm_size = "Standard_DS1_v2"

    storage_image_reference {
	publisher = "kemptech"
	offer = "vlm-azure"
	sku = "freeloadmaster"
	version = "latest"
    }

    storage_os_disk {
        name = "myosdisk1"
        vhd_uri = "${azurerm_storage_account.test.primary_blob_endpoint}${azurerm_storage_container.test.name}/myosdisk1.vhd"
        caching = "ReadWrite"
        create_option = "FromImage"
        disk_size_gb = "45"
    }

    os_profile {
	computer_name = "hostname%d"
	admin_username = "testadmin"
	admin_password = "Password1234!"
    }

    os_profile_linux_config {
	disable_password_authentication = false
    }

    plan {
        name = "freeloadmaster"
        publisher = "kemptech"
        product = "vlm-azure"
    }

    tags {
    	environment = "Production"
    	cost-center = "Ops"
    }
}
`

var testAccAzureRMVirtualMachine_linuxMachineWithSSH = `
resource "azurerm_resource_group" "test" {
    name = "acctestrg%s"
    location = "southcentralus"
}

resource "azurerm_virtual_network" "test" {
    name = "acctvn%s"
    address_space = ["10.0.0.0/16"]
    location = "southcentralus"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_subnet" "test" {
    name = "acctsub%s"
    resource_group_name = "${azurerm_resource_group.test.name}"
    virtual_network_name = "${azurerm_virtual_network.test.name}"
    address_prefix = "10.0.2.0/24"
}

resource "azurerm_network_interface" "test" {
    name = "acctni%s"
    location = "southcentralus"
    resource_group_name = "${azurerm_resource_group.test.name}"

    ip_configuration {
    	name = "testconfiguration1"
    	subnet_id = "${azurerm_subnet.test.id}"
    	private_ip_address_allocation = "dynamic"
    }
}

resource "azurerm_storage_account" "test" {
    name = "accsa%s"
    resource_group_name = "${azurerm_resource_group.test.name}"
    location = "southcentralus"
    account_type = "Standard_LRS"
}

resource "azurerm_storage_container" "test" {
    name = "vhds"
    resource_group_name = "${azurerm_resource_group.test.name}"
    storage_account_name = "${azurerm_storage_account.test.name}"
    container_access_type = "private"
}

resource "azurerm_virtual_machine" "test" {
    name = "acctvm%s"
    location = "southcentralus"
    resource_group_name = "${azurerm_resource_group.test.name}"
    network_interface_ids = ["${azurerm_network_interface.test.id}"]
    vm_size = "Standard_A0"

    storage_image_reference {
        publisher = "Canonical"
        offer = "UbuntuServer"
        sku = "14.04.2-LTS"
        version = "latest"
    }

    storage_os_disk {
        name = "myosdisk1"
        vhd_uri = "${azurerm_storage_account.test.primary_blob_endpoint}${azurerm_storage_container.test.name}/myosdisk1.vhd"
        caching = "ReadWrite"
        create_option = "FromImage"
        disk_size_gb = "45"
    }

    os_profile {
        computer_name = "hostname%s"
        admin_username = "testadmin"
        admin_password = "Password1234!"
    }

    os_profile_linux_config {
	    disable_password_authentication = true
        ssh_keys {
            path = "/home/testadmin/.ssh/authorized_keys"
            key_data = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAAgQCfGyt5W1eJVpDIxlyvAWO594j/azEGohmlxYe7mgSfmUCWjuzILI6nHuHbxhpBDIZJhQ+JAeduXpii61dmThbI89ghGMhzea0OlT3p12e093zqa4goB9g40jdNKmJArER3pMVqs6hmv8y3GlUNkMDSmuoyI8AYzX4n26cUKZbwXQ== mk@mk3"
        }
    }
}
`

var testAccAzureRMVirtualMachine_linuxMachineWithSSHRemoved = `
resource "azurerm_resource_group" "test" {
    name = "acctestrg%s"
    location = "southcentralus"
}

resource "azurerm_virtual_network" "test" {
    name = "acctvn%s"
    address_space = ["10.0.0.0/16"]
    location = "southcentralus"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_subnet" "test" {
    name = "acctsub%s"
    resource_group_name = "${azurerm_resource_group.test.name}"
    virtual_network_name = "${azurerm_virtual_network.test.name}"
    address_prefix = "10.0.2.0/24"
}

resource "azurerm_network_interface" "test" {
    name = "acctni%s"
    location = "southcentralus"
    resource_group_name = "${azurerm_resource_group.test.name}"

    ip_configuration {
    	name = "testconfiguration1"
    	subnet_id = "${azurerm_subnet.test.id}"
    	private_ip_address_allocation = "dynamic"
    }
}

resource "azurerm_storage_account" "test" {
    name = "accsa%s"
    resource_group_name = "${azurerm_resource_group.test.name}"
    location = "southcentralus"
    account_type = "Standard_LRS"
}

resource "azurerm_storage_container" "test" {
    name = "vhds"
    resource_group_name = "${azurerm_resource_group.test.name}"
    storage_account_name = "${azurerm_storage_account.test.name}"
    container_access_type = "private"
}

resource "azurerm_virtual_machine" "test" {
    name = "acctvm%s"
    location = "southcentralus"
    resource_group_name = "${azurerm_resource_group.test.name}"
    network_interface_ids = ["${azurerm_network_interface.test.id}"]
    vm_size = "Standard_A0"

    storage_image_reference {
        publisher = "Canonical"
        offer = "UbuntuServer"
        sku = "14.04.2-LTS"
        version = "latest"
    }

    storage_os_disk {
        name = "myosdisk1"
        vhd_uri = "${azurerm_storage_account.test.primary_blob_endpoint}${azurerm_storage_container.test.name}/myosdisk1.vhd"
        caching = "ReadWrite"
        create_option = "FromImage"
        disk_size_gb = "45"
    }

    os_profile {
        computer_name = "hostname%s"
        admin_username = "testadmin"
        admin_password = "Password1234!"
    }

    os_profile_linux_config {
	    disable_password_authentication = true
    }
}
`

var testAccAzureRMVirtualMachine_primaryNetworkInterfaceId = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
}

resource "azurerm_virtual_network" "test" {
    name = "acctvn-%d"
    address_space = ["10.0.0.0/16"]
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_subnet" "test" {
    name = "acctsub-%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    virtual_network_name = "${azurerm_virtual_network.test.name}"
    address_prefix = "10.0.2.0/24"
}

resource "azurerm_network_interface" "test" {
    name = "acctni-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"

    ip_configuration {
    	name = "testconfiguration1"
    	subnet_id = "${azurerm_subnet.test.id}"
    	private_ip_address_allocation = "dynamic"
    }
}

resource "azurerm_network_interface" "test2" {
    name = "acctni2-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"

    ip_configuration {
    	name = "testconfiguration2"
    	subnet_id = "${azurerm_subnet.test.id}"
    	private_ip_address_allocation = "dynamic"
    }
}

resource "azurerm_storage_account" "test" {
    name = "accsa%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    location = "westus"
    account_type = "Standard_LRS"

    tags {
        environment = "staging"
    }
}

resource "azurerm_storage_container" "test" {
    name = "vhds"
    resource_group_name = "${azurerm_resource_group.test.name}"
    storage_account_name = "${azurerm_storage_account.test.name}"
    container_access_type = "private"
}

resource "azurerm_virtual_machine" "test" {
    name = "acctvm-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    network_interface_ids = ["${azurerm_network_interface.test.id}","${azurerm_network_interface.test2.id}"]
    primary_network_interface_id = "${azurerm_network_interface.test.id}"
    vm_size = "Standard_A3"

    storage_image_reference {
	publisher = "Canonical"
	offer = "UbuntuServer"
	sku = "14.04.2-LTS"
	version = "latest"
    }

    storage_os_disk {
        name = "myosdisk1"
        vhd_uri = "${azurerm_storage_account.test.primary_blob_endpoint}${azurerm_storage_container.test.name}/myosdisk1.vhd"
        caching = "ReadWrite"
        create_option = "FromImage"
        disk_size_gb = "45"
    }

    os_profile {
	computer_name = "hostname"
	admin_username = "testadmin"
	admin_password = "Password1234!"
    }

    os_profile_linux_config {
	disable_password_authentication = false
    }

    tags {
    	environment = "Production"
    	cost-center = "Ops"
    }
}
`
