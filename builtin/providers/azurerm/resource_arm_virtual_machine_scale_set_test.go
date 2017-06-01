package azurerm

import (
	"fmt"
	"net/http"
	"regexp"
	"testing"

	"github.com/Azure/azure-sdk-for-go/arm/compute"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAzureRMVirtualMachineScaleSet_basic(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMVirtualMachineScaleSet_basic, ri, ri, ri, ri, ri, ri, ri, ri)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMVirtualMachineScaleSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualMachineScaleSetExists("azurerm_virtual_machine_scale_set.test"),

					// single placement group should default to true
					testCheckAzureRMVirtualMachineScaleSetSinglePlacementGroup("azurerm_virtual_machine_scale_set.test", true),
				),
			},
		},
	})
}

func TestAccAzureRMVirtualMachineScaleSet_singlePlacementGroupFalse(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMVirtualMachineScaleSet_singlePlacementGroupFalse, ri)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMVirtualMachineScaleSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualMachineScaleSetExists("azurerm_virtual_machine_scale_set.test"),
					testCheckAzureRMVirtualMachineScaleSetSinglePlacementGroup("azurerm_virtual_machine_scale_set.test", false),
				),
			},
		},
	})
}

func TestAccAzureRMVirtualMachineScaleSet_linuxUpdated(t *testing.T) {
	resourceName := "azurerm_virtual_machine_scale_set.test"
	ri := acctest.RandInt()
	config := testAccAzureRMVirtualMachineScaleSet_linux(ri)
	updatedConfig := testAccAzureRMVirtualMachineScaleSet_linuxUpdated(ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMVirtualMachineScaleSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualMachineScaleSetExists(resourceName),
				),
			},
			{
				Config: updatedConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualMachineScaleSetExists(resourceName),
				),
			},
		},
	})
}

func TestAccAzureRMVirtualMachineScaleSet_basicLinux_managedDisk(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMVirtualMachineScaleSet_basicLinux_managedDisk, ri, ri, ri, ri, ri, ri)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMVirtualMachineScaleSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualMachineScaleSetExists("azurerm_virtual_machine_scale_set.test"),
				),
			},
		},
	})
}

func TestAccAzureRMVirtualMachineScaleSet_basicLinux_disappears(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMVirtualMachineScaleSet_basic, ri, ri, ri, ri, ri, ri, ri, ri)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMVirtualMachineScaleSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualMachineScaleSetExists("azurerm_virtual_machine_scale_set.test"),
					testCheckAzureRMVirtualMachineScaleSetDisappears("azurerm_virtual_machine_scale_set.test"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccAzureRMVirtualMachineScaleSet_loadBalancer(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMVirtualMachineScaleSetLoadbalancerTemplate, ri, ri, ri, ri, ri, ri, ri)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMVirtualMachineScaleSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualMachineScaleSetExists("azurerm_virtual_machine_scale_set.test"),
					testCheckAzureRMVirtualMachineScaleSetHasLoadbalancer("azurerm_virtual_machine_scale_set.test"),
				),
			},
		},
	})
}

func TestAccAzureRMVirtualMachineScaleSet_loadBalancerManagedDataDisks(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMVirtualMachineScaleSetLoadbalancerTemplateManagedDataDisks, ri, ri, ri, ri, ri, ri)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMVirtualMachineScaleSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualMachineScaleSetExists("azurerm_virtual_machine_scale_set.test"),
					testCheckAzureRMVirtualMachineScaleSetHasDataDisks("azurerm_virtual_machine_scale_set.test"),
				),
			},
		},
	})
}

func TestAccAzureRMVirtualMachineScaleSet_overprovision(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMVirtualMachineScaleSetOverprovisionTemplate, ri, ri, ri, ri, ri, ri)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMVirtualMachineScaleSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualMachineScaleSetExists("azurerm_virtual_machine_scale_set.test"),
					testCheckAzureRMVirtualMachineScaleSetOverprovision("azurerm_virtual_machine_scale_set.test"),
				),
			},
		},
	})
}

func TestAccAzureRMVirtualMachineScaleSet_extension(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMVirtualMachineScaleSetExtensionTemplate, ri, ri, ri, ri, ri, ri)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMVirtualMachineScaleSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualMachineScaleSetExists("azurerm_virtual_machine_scale_set.test"),
					testCheckAzureRMVirtualMachineScaleSetExtension("azurerm_virtual_machine_scale_set.test"),
				),
			},
		},
	})
}

func TestAccAzureRMVirtualMachineScaleSet_multipleExtensions(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMVirtualMachineScaleSetMultipleExtensionsTemplate, ri, ri, ri, ri, ri, ri)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMVirtualMachineScaleSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualMachineScaleSetExists("azurerm_virtual_machine_scale_set.test"),
					testCheckAzureRMVirtualMachineScaleSetExtension("azurerm_virtual_machine_scale_set.test"),
				),
			},
		},
	})
}

func TestAccAzureRMVirtualMachineScaleSet_osDiskTypeConflict(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMVirtualMachineScaleSet_osDiskTypeConflict, ri, ri, ri, ri, ri, ri, ri)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMVirtualMachineScaleSetDestroy,
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile("Conflict between `vhd_containers`"),
				//Use below code instead once GH-13019 has been merged
				//ExpectError: regexp.MustCompile("conflicts with storage_profile_os_disk.0.vhd_containers"),
			},
		},
	})
}

func TestAccAzureRMVirtualMachineScaleSet_NonStandardCasing(t *testing.T) {
	ri := acctest.RandInt()
	config := testAccAzureRMVirtualMachineScaleSetNonStandardCasing(ri)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMVirtualMachineScaleSetDestroy,
		Steps: []resource.TestStep{

			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualMachineScaleSetExists("azurerm_virtual_machine_scale_set.test"),
				),
			},

			resource.TestStep{
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func testGetAzureRMVirtualMachineScaleSet(s *terraform.State, resourceName string) (result *compute.VirtualMachineScaleSet, err error) {
	// Ensure we have enough information in state to look up in API
	rs, ok := s.RootModule().Resources[resourceName]
	if !ok {
		return nil, fmt.Errorf("Not found: %s", resourceName)
	}

	// Name of the actual scale set
	name := rs.Primary.Attributes["name"]

	resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
	if !hasResourceGroup {
		return nil, fmt.Errorf("Bad: no resource group found in state for virtual machine: scale set %s", name)
	}

	conn := testAccProvider.Meta().(*ArmClient).vmScaleSetClient

	vmss, err := conn.Get(resourceGroup, name)
	if err != nil {
		return nil, fmt.Errorf("Bad: Get on vmScaleSetClient: %s", err)
	}

	if vmss.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("Bad: VirtualMachineScaleSet %q (resource group: %q) does not exist", name, resourceGroup)
	}

	return &vmss, err
}

func testCheckAzureRMVirtualMachineScaleSetExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, err := testGetAzureRMVirtualMachineScaleSet(s, name)
		return err
	}
}

func testCheckAzureRMVirtualMachineScaleSetDisappears(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		name := rs.Primary.Attributes["name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for virtual machine: scale set %s", name)
		}

		conn := testAccProvider.Meta().(*ArmClient).vmScaleSetClient

		_, error := conn.Delete(resourceGroup, name, make(chan struct{}))
		err := <-error
		if err != nil {
			return fmt.Errorf("Bad: Delete on vmScaleSetClient: %s", err)
		}

		return nil
	}
}

func testCheckAzureRMVirtualMachineScaleSetDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*ArmClient).vmScaleSetClient

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_virtual_machine_scale_set" {
			continue
		}

		name := rs.Primary.Attributes["name"]
		resourceGroup := rs.Primary.Attributes["resource_group_name"]

		resp, err := conn.Get(resourceGroup, name)

		if err != nil {
			return nil
		}

		if resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("Virtual Machine Scale Set still exists:\n%#v", resp.VirtualMachineScaleSetProperties)
		}
	}

	return nil
}

func testCheckAzureRMVirtualMachineScaleSetHasLoadbalancer(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resp, err := testGetAzureRMVirtualMachineScaleSet(s, name)
		if err != nil {
			return err
		}

		n := resp.VirtualMachineProfile.NetworkProfile.NetworkInterfaceConfigurations
		if n == nil || len(*n) == 0 {
			return fmt.Errorf("Bad: Could not get network interface configurations for scale set %v", name)
		}

		ip := (*n)[0].IPConfigurations
		if ip == nil || len(*ip) == 0 {
			return fmt.Errorf("Bad: Could not get ip configurations for scale set %v", name)
		}

		pools := (*ip)[0].LoadBalancerBackendAddressPools
		if pools == nil || len(*pools) == 0 {
			return fmt.Errorf("Bad: Load balancer backend pools is empty for scale set %v", name)
		}

		return nil
	}
}

func testCheckAzureRMVirtualMachineScaleSetOverprovision(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resp, err := testGetAzureRMVirtualMachineScaleSet(s, name)
		if err != nil {
			return err
		}

		if *resp.Overprovision {
			return fmt.Errorf("Bad: Overprovision should have been false for scale set %v", name)
		}

		return nil
	}
}

func testCheckAzureRMVirtualMachineScaleSetSinglePlacementGroup(name string, expectedSinglePlacementGroup bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resp, err := testGetAzureRMVirtualMachineScaleSet(s, name)
		if err != nil {
			return err
		}

		if *resp.SinglePlacementGroup != expectedSinglePlacementGroup {
			return fmt.Errorf("Bad: Overprovision should have been %t for scale set %v", expectedSinglePlacementGroup, name)
		}

		return nil
	}
}

func testCheckAzureRMVirtualMachineScaleSetExtension(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resp, err := testGetAzureRMVirtualMachineScaleSet(s, name)
		if err != nil {
			return err
		}

		n := resp.VirtualMachineProfile.ExtensionProfile.Extensions
		if n == nil || len(*n) == 0 {
			return fmt.Errorf("Bad: Could not get extensions for scale set %v", name)
		}

		return nil
	}
}

func testCheckAzureRMVirtualMachineScaleSetHasDataDisks(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		name := rs.Primary.Attributes["name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for virtual machine: scale set %s", name)
		}

		conn := testAccProvider.Meta().(*ArmClient).vmScaleSetClient
		resp, err := conn.Get(resourceGroup, name)
		if err != nil {
			return fmt.Errorf("Bad: Get on vmScaleSetClient: %s", err)
		}

		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("Bad: VirtualMachineScaleSet %q (resource group: %q) does not exist", name, resourceGroup)
		}

		storageProfile := resp.VirtualMachineProfile.StorageProfile.DataDisks
		if storageProfile == nil || len(*storageProfile) == 0 {
			return fmt.Errorf("Bad: Could not get data disks configurations for scale set %v", name)
		}

		return nil
	}
}

var testAccAzureRMVirtualMachineScaleSet_basic = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US 2"
}

resource "azurerm_virtual_network" "test" {
    name = "acctvn-%d"
    address_space = ["10.0.0.0/16"]
    location = "West US 2"
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
    location = "West US 2"
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
    location = "West US 2"
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

resource "azurerm_virtual_machine_scale_set" "test" {
  name = "acctvmss-%d"
  location = "West US 2"
  resource_group_name = "${azurerm_resource_group.test.name}"
  upgrade_policy_mode = "Manual"

  sku {
    name = "Standard_D1_v2"
    tier = "Standard"
    capacity = 2
  }

  os_profile {
    computer_name_prefix = "testvm-%d"
    admin_username = "myadmin"
    admin_password = "Passwword1234"
  }

  network_profile {
      name = "TestNetworkProfile-%d"
      primary = true
      ip_configuration {
        name = "TestIPConfiguration"
        subnet_id = "${azurerm_subnet.test.id}"
      }
  }

  storage_profile_os_disk {
    name = "osDiskProfile"
    caching       = "ReadWrite"
    create_option = "FromImage"
    vhd_containers = ["${azurerm_storage_account.test.primary_blob_endpoint}${azurerm_storage_container.test.name}"]
  }

  storage_profile_image_reference {
    publisher = "Canonical"
    offer     = "UbuntuServer"
    sku       = "16.04-LTS"
    version   = "latest"
  }
}
`

var testAccAzureRMVirtualMachineScaleSet_singlePlacementGroupFalse = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%[1]d"
    location = "West US 2"
}

resource "azurerm_virtual_network" "test" {
    name = "acctvn-%[1]d"
    address_space = ["10.0.0.0/16"]
    location = "West US 2"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_subnet" "test" {
    name = "acctsub-%[1]d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    virtual_network_name = "${azurerm_virtual_network.test.name}"
    address_prefix = "10.0.2.0/24"
}

resource "azurerm_network_interface" "test" {
    name = "acctni-%[1]d"
    location = "West US 2"
    resource_group_name = "${azurerm_resource_group.test.name}"

    ip_configuration {
    	name = "testconfiguration1"
    	subnet_id = "${azurerm_subnet.test.id}"
    	private_ip_address_allocation = "dynamic"
    }
}

resource "azurerm_storage_account" "test" {
    name = "accsa%[1]d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    location = "West US 2"
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

resource "azurerm_virtual_machine_scale_set" "test" {
  name = "acctvmss-%[1]d"
  location = "West US 2"
  resource_group_name = "${azurerm_resource_group.test.name}"
  upgrade_policy_mode = "Manual"
  single_placement_group = false

  sku {
    name = "Standard_D1_v2"
    tier = "Standard"
    capacity = 2
  }

  os_profile {
    computer_name_prefix = "testvm-%[1]d"
    admin_username = "myadmin"
    admin_password = "Passwword1234"
  }

  network_profile {
      name = "TestNetworkProfile-%[1]d"
      primary = true
      ip_configuration {
        name = "TestIPConfiguration"
        subnet_id = "${azurerm_subnet.test.id}"
      }
  }

  storage_profile_os_disk {
    name = ""
    caching       = "ReadWrite"
    create_option = "FromImage"
    managed_disk_type = "Standard_LRS"
  }

  storage_profile_image_reference {
    publisher = "Canonical"
    offer     = "UbuntuServer"
    sku       = "16.04-LTS"
    version   = "latest"
  }
}
`

func testAccAzureRMVirtualMachineScaleSet_linux(rInt int) string {
	return fmt.Sprintf(`
	resource "azurerm_resource_group" "test" {
  name     = "acctestrg-%d"
  location = "West Europe"
}
resource "azurerm_virtual_network" "test" {
  name                = "acctestvn-%d"
  resource_group_name = "${azurerm_resource_group.test.name}"
  location            = "${azurerm_resource_group.test.location}"
  address_space       = ["10.0.0.0/8"]
}
resource "azurerm_subnet" "test" {
  name                 = "acctestsn-%d"
  resource_group_name  = "${azurerm_resource_group.test.name}"
  virtual_network_name = "${azurerm_virtual_network.test.name}"
  address_prefix       = "10.0.1.0/24"
}
resource "azurerm_storage_account" "test" {
  name                = "accsa%d"
  resource_group_name = "${azurerm_resource_group.test.name}"
  location            = "${azurerm_resource_group.test.location}"
  account_type        = "Standard_LRS"
}
resource "azurerm_storage_container" "test" {
  name                  = "acctestsc-%d"
  resource_group_name   = "${azurerm_resource_group.test.name}"
  storage_account_name  = "${azurerm_storage_account.test.name}"
  container_access_type = "private"
}
resource "azurerm_public_ip" "test" {
  name                         = "acctestpip-%d"
  resource_group_name          = "${azurerm_resource_group.test.name}"
  location                     = "${azurerm_resource_group.test.location}"
  public_ip_address_allocation = "static"
}
resource "azurerm_lb" "test" {
  name                = "acctestlb-%d"
  resource_group_name = "${azurerm_resource_group.test.name}"
  location            = "${azurerm_resource_group.test.location}"
  frontend_ip_configuration {
    name                 = "ip-address"
    public_ip_address_id = "${azurerm_public_ip.test.id}"
  }
}
resource "azurerm_lb_backend_address_pool" "test" {
  name                = "acctestbap-%d"
  resource_group_name = "${azurerm_resource_group.test.name}"
  loadbalancer_id     = "${azurerm_lb.test.id}"
}
resource "azurerm_virtual_machine_scale_set" "test" {
  name                = "acctestvmss-%d"
  resource_group_name = "${azurerm_resource_group.test.name}"
  location            = "${azurerm_resource_group.test.location}"
  upgrade_policy_mode = "Automatic"
  sku {
    name     = "Standard_A0"
    tier     = "Standard"
    capacity = "1"
  }
  os_profile {
    computer_name_prefix = "prefix"
    admin_username       = "ubuntu"
    admin_password       = "password"
    custom_data          = "custom data!"
  }
  os_profile_linux_config {
    disable_password_authentication = true
    ssh_keys {
      path     = "/home/ubuntu/.ssh/authorized_keys"
      key_data = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDCsTcryUl51Q2VSEHqDRNmceUFo55ZtcIwxl2QITbN1RREti5ml/VTytC0yeBOvnZA4x4CFpdw/lCDPk0yrH9Ei5vVkXmOrExdTlT3qI7YaAzj1tUVlBd4S6LX1F7y6VLActvdHuDDuXZXzCDd/97420jrDfWZqJMlUK/EmCE5ParCeHIRIvmBxcEnGfFIsw8xQZl0HphxWOtJil8qsUWSdMyCiJYYQpMoMliO99X40AUc4/AlsyPyT5ddbKk08YrZ+rKDVHF7o29rh4vi5MmHkVgVQHKiKybWlHq+b71gIAUQk9wrJxD+dqt4igrmDSpIjfjwnd+l5UIn5fJSO5DYV4YT/4hwK7OKmuo7OFHD0WyY5YnkYEMtFgzemnRBdE8ulcT60DQpVgRMXFWHvhyCWy0L6sgj1QWDZlLpvsIvNfHsyhKFMG1frLnMt/nP0+YCcfg+v1JYeCKjeoJxB8DWcRBsjzItY0CGmzP8UYZiYKl/2u+2TgFS5r7NWH11bxoUzjKdaa1NLw+ieA8GlBFfCbfWe6YVB9ggUte4VtYFMZGxOjS2bAiYtfgTKFJv+XqORAwExG6+G2eDxIDyo80/OA9IG7Xv/jwQr7D6KDjDuULFcN/iTxuttoKrHeYz1hf5ZQlBdllwJHYx6fK2g8kha6r2JIQKocvsAXiiONqSfw== hello@world.com"
    }
  }
  network_profile {
    name    = "TestNetworkProfile"
    primary = true
    ip_configuration {
      name                                   = "TestIPConfiguration"
      subnet_id                              = "${azurerm_subnet.test.id}"
      load_balancer_backend_address_pool_ids = ["${azurerm_lb_backend_address_pool.test.id}"]
    }
  }
  storage_profile_os_disk {
    name           = "osDiskProfile"
    caching        = "ReadWrite"
    create_option  = "FromImage"
    os_type        = "linux"
    vhd_containers = ["${azurerm_storage_account.test.primary_blob_endpoint}${azurerm_storage_container.test.name}"]
  }
  storage_profile_image_reference {
    publisher = "Canonical"
    offer     = "UbuntuServer"
    sku       = "14.04.2-LTS"
    version   = "latest"
  }
}
`, rInt, rInt, rInt, rInt, rInt, rInt, rInt, rInt, rInt)
}

func testAccAzureRMVirtualMachineScaleSet_linuxUpdated(rInt int) string {
	return fmt.Sprintf(`
	resource "azurerm_resource_group" "test" {
  name     = "acctestrg-%d"
  location = "West Europe"
}
resource "azurerm_virtual_network" "test" {
  name                = "acctestvn-%d"
  resource_group_name = "${azurerm_resource_group.test.name}"
  location            = "${azurerm_resource_group.test.location}"
  address_space       = ["10.0.0.0/8"]
}
resource "azurerm_subnet" "test" {
  name                 = "acctestsn-%d"
  resource_group_name  = "${azurerm_resource_group.test.name}"
  virtual_network_name = "${azurerm_virtual_network.test.name}"
  address_prefix       = "10.0.1.0/24"
}
resource "azurerm_storage_account" "test" {
  name                = "accsa%d"
  resource_group_name = "${azurerm_resource_group.test.name}"
  location            = "${azurerm_resource_group.test.location}"
  account_type        = "Standard_LRS"
}
resource "azurerm_storage_container" "test" {
  name                  = "acctestsc-%d"
  resource_group_name   = "${azurerm_resource_group.test.name}"
  storage_account_name  = "${azurerm_storage_account.test.name}"
  container_access_type = "private"
}
resource "azurerm_public_ip" "test" {
  name                         = "acctestpip-%d"
  resource_group_name          = "${azurerm_resource_group.test.name}"
  location                     = "${azurerm_resource_group.test.location}"
  public_ip_address_allocation = "static"
}
resource "azurerm_lb" "test" {
  name                = "acctestlb-%d"
  resource_group_name = "${azurerm_resource_group.test.name}"
  location            = "${azurerm_resource_group.test.location}"
  frontend_ip_configuration {
    name                 = "ip-address"
    public_ip_address_id = "${azurerm_public_ip.test.id}"
  }
}
resource "azurerm_lb_backend_address_pool" "test" {
  name                = "acctestbap-%d"
  resource_group_name = "${azurerm_resource_group.test.name}"
  loadbalancer_id     = "${azurerm_lb.test.id}"
}
resource "azurerm_virtual_machine_scale_set" "test" {
  name                = "acctestvmss-%d"
  resource_group_name = "${azurerm_resource_group.test.name}"
  location            = "${azurerm_resource_group.test.location}"
  upgrade_policy_mode = "Automatic"
  sku {
    name     = "Standard_A0"
    tier     = "Standard"
    capacity = "1"
  }
  os_profile {
    computer_name_prefix = "prefix"
    admin_username       = "ubuntu"
    admin_password       = "password"
    custom_data          = "custom data!"
  }
  os_profile_linux_config {
    disable_password_authentication = true
    ssh_keys {
      path     = "/home/ubuntu/.ssh/authorized_keys"
      key_data = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDCsTcryUl51Q2VSEHqDRNmceUFo55ZtcIwxl2QITbN1RREti5ml/VTytC0yeBOvnZA4x4CFpdw/lCDPk0yrH9Ei5vVkXmOrExdTlT3qI7YaAzj1tUVlBd4S6LX1F7y6VLActvdHuDDuXZXzCDd/97420jrDfWZqJMlUK/EmCE5ParCeHIRIvmBxcEnGfFIsw8xQZl0HphxWOtJil8qsUWSdMyCiJYYQpMoMliO99X40AUc4/AlsyPyT5ddbKk08YrZ+rKDVHF7o29rh4vi5MmHkVgVQHKiKybWlHq+b71gIAUQk9wrJxD+dqt4igrmDSpIjfjwnd+l5UIn5fJSO5DYV4YT/4hwK7OKmuo7OFHD0WyY5YnkYEMtFgzemnRBdE8ulcT60DQpVgRMXFWHvhyCWy0L6sgj1QWDZlLpvsIvNfHsyhKFMG1frLnMt/nP0+YCcfg+v1JYeCKjeoJxB8DWcRBsjzItY0CGmzP8UYZiYKl/2u+2TgFS5r7NWH11bxoUzjKdaa1NLw+ieA8GlBFfCbfWe6YVB9ggUte4VtYFMZGxOjS2bAiYtfgTKFJv+XqORAwExG6+G2eDxIDyo80/OA9IG7Xv/jwQr7D6KDjDuULFcN/iTxuttoKrHeYz1hf5ZQlBdllwJHYx6fK2g8kha6r2JIQKocvsAXiiONqSfw== hello@world.com"
    }
  }
  network_profile {
    name    = "TestNetworkProfile"
    primary = true
    ip_configuration {
      name                                   = "TestIPConfiguration"
      subnet_id                              = "${azurerm_subnet.test.id}"
      load_balancer_backend_address_pool_ids = ["${azurerm_lb_backend_address_pool.test.id}"]
    }
  }
  storage_profile_os_disk {
    name           = "osDiskProfile"
    caching        = "ReadWrite"
    create_option  = "FromImage"
    os_type        = "linux"
    vhd_containers = ["${azurerm_storage_account.test.primary_blob_endpoint}${azurerm_storage_container.test.name}"]
  }
  storage_profile_image_reference {
    publisher = "Canonical"
    offer     = "UbuntuServer"
    sku       = "14.04.2-LTS"
    version   = "latest"
  }
  tags {
    ThisIs = "a test"
  }
}
`, rInt, rInt, rInt, rInt, rInt, rInt, rInt, rInt, rInt)
}

var testAccAzureRMVirtualMachineScaleSet_basicLinux_managedDisk = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US 2"
}

resource "azurerm_virtual_network" "test" {
    name = "acctvn-%d"
    address_space = ["10.0.0.0/16"]
    location = "West US 2"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_subnet" "test" {
    name = "acctsub-%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    virtual_network_name = "${azurerm_virtual_network.test.name}"
    address_prefix = "10.0.2.0/24"
}

resource "azurerm_virtual_machine_scale_set" "test" {
  name = "acctvmss-%d"
  location = "West US 2"
  resource_group_name = "${azurerm_resource_group.test.name}"
  upgrade_policy_mode = "Manual"

  sku {
    name = "Standard_D1_v2"
    tier = "Standard"
    capacity = 2
  }

  os_profile {
    computer_name_prefix = "testvm-%d"
    admin_username = "myadmin"
    admin_password = "Passwword1234"
  }

  network_profile {
      name = "TestNetworkProfile-%d"
      primary = true
      ip_configuration {
        name = "TestIPConfiguration"
        subnet_id = "${azurerm_subnet.test.id}"
      }
  }

  storage_profile_os_disk {
	name 		  = ""
    caching       = "ReadWrite"
    create_option = "FromImage"
    managed_disk_type = "Standard_LRS"
  }

  storage_profile_image_reference {
    publisher = "Canonical"
    offer     = "UbuntuServer"
    sku       = "16.04-LTS"
    version   = "latest"
  }
}
`

var testAccAzureRMVirtualMachineScaleSetLoadbalancerTemplate = `
resource "azurerm_resource_group" "test" {
    name 	 = "acctestrg-%d"
    location = "southcentralus"
}

resource "azurerm_virtual_network" "test" {
    name 		        = "acctvn-%d"
    address_space       = ["10.0.0.0/16"]
    location            = "southcentralus"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_subnet" "test" {
    name                 = "acctsub-%d"
    resource_group_name  = "${azurerm_resource_group.test.name}"
    virtual_network_name = "${azurerm_virtual_network.test.name}"
    address_prefix       = "10.0.2.0/24"
}

resource "azurerm_storage_account" "test" {
    name                = "accsa%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    location            = "southcentralus"
    account_type        = "Standard_LRS"
}

resource "azurerm_storage_container" "test" {
    name                  = "vhds"
    resource_group_name   = "${azurerm_resource_group.test.name}"
    storage_account_name  = "${azurerm_storage_account.test.name}"
    container_access_type = "private"
}

resource "azurerm_lb" "test" {
    name                = "acctestlb-%d"
    location            = "southcentralus"
    resource_group_name = "${azurerm_resource_group.test.name}"

    frontend_ip_configuration {
        name                          = "default"
        subnet_id                     = "${azurerm_subnet.test.id}"
        private_ip_address_allocation = "Dynamic"
    }
}

resource "azurerm_lb_backend_address_pool" "test" {
    name                = "test"
    resource_group_name = "${azurerm_resource_group.test.name}"
    location            = "southcentralus"
    loadbalancer_id     = "${azurerm_lb.test.id}"
}

resource "azurerm_lb_nat_pool" "test" {
  resource_group_name = "${azurerm_resource_group.test.name}"
  name                           = "ssh"
  loadbalancer_id                = "${azurerm_lb.test.id}"
  protocol                       = "Tcp"
  frontend_port_start            = 50000
  frontend_port_end              = 50119
  backend_port                   = 22
  frontend_ip_configuration_name = "default"
}

resource "azurerm_virtual_machine_scale_set" "test" {
  	name                = "acctvmss-%d"
  	location            = "southcentralus"
  	resource_group_name = "${azurerm_resource_group.test.name}"
  	upgrade_policy_mode = "Manual"

  	sku {
		name     = "Standard_D1_v2"
    	tier     = "Standard"
    	capacity = 1
	}

  	os_profile {
    	computer_name_prefix = "testvm-%d"
    	admin_username = "myadmin"
    	admin_password = "Passwword1234"
  	}

  	network_profile {
      	name    = "TestNetworkProfile"
      	primary = true
      	ip_configuration {
        	name                                   = "TestIPConfiguration"
        	subnet_id                              = "${azurerm_subnet.test.id}"
			load_balancer_backend_address_pool_ids = [ "${azurerm_lb_backend_address_pool.test.id}" ]
		    load_balancer_inbound_nat_rules_ids = ["${azurerm_lb_nat_pool.test.id}"]
      	}
  	}

  	storage_profile_os_disk {
    	name 		   = "os-disk"
    	caching        = "ReadWrite"
    	create_option  = "FromImage"
    	vhd_containers = [ "${azurerm_storage_account.test.primary_blob_endpoint}${azurerm_storage_container.test.name}" ]
  	}

  	storage_profile_image_reference {
    	publisher = "Canonical"
    	offer     = "UbuntuServer"
    	sku       = "16.04-LTS"
    	version   = "latest"
  	}
}
`

var testAccAzureRMVirtualMachineScaleSetOverprovisionTemplate = `
resource "azurerm_resource_group" "test" {
    name 	 = "acctestrg-%d"
    location = "southcentralus"
}

resource "azurerm_virtual_network" "test" {
    name 		        = "acctvn-%d"
    address_space       = ["10.0.0.0/16"]
    location            = "southcentralus"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_subnet" "test" {
    name                 = "acctsub-%d"
    resource_group_name  = "${azurerm_resource_group.test.name}"
    virtual_network_name = "${azurerm_virtual_network.test.name}"
    address_prefix       = "10.0.2.0/24"
}

resource "azurerm_storage_account" "test" {
    name                = "accsa%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    location            = "southcentralus"
    account_type        = "Standard_LRS"
}

resource "azurerm_storage_container" "test" {
    name                  = "vhds"
    resource_group_name   = "${azurerm_resource_group.test.name}"
    storage_account_name  = "${azurerm_storage_account.test.name}"
    container_access_type = "private"
}

resource "azurerm_virtual_machine_scale_set" "test" {
  	name                = "acctvmss-%d"
  	location            = "southcentralus"
  	resource_group_name = "${azurerm_resource_group.test.name}"
  	upgrade_policy_mode = "Manual"
	overprovision       = false

  	sku {
		name     = "Standard_D1_v2"
    	tier     = "Standard"
    	capacity = 1
	}

  	os_profile {
    	computer_name_prefix = "testvm-%d"
    	admin_username = "myadmin"
    	admin_password = "Passwword1234"
  	}

  	network_profile {
      	name    = "TestNetworkProfile"
      	primary = true
      	ip_configuration {
        	name	  = "TestIPConfiguration"
        	subnet_id = "${azurerm_subnet.test.id}"
      	}
  	}

  	storage_profile_os_disk {
    	name 		   = "os-disk"
    	caching        = "ReadWrite"
    	create_option  = "FromImage"
    	vhd_containers = [ "${azurerm_storage_account.test.primary_blob_endpoint}${azurerm_storage_container.test.name}" ]
  	}

  	storage_profile_image_reference {
    	publisher = "Canonical"
    	offer     = "UbuntuServer"
    	sku       = "16.04-LTS"
    	version   = "latest"
  	}
}
`

var testAccAzureRMVirtualMachineScaleSetExtensionTemplate = `
resource "azurerm_resource_group" "test" {
    name 	 = "acctestrg-%d"
    location = "southcentralus"
}

resource "azurerm_virtual_network" "test" {
    name 		        = "acctvn-%d"
    address_space       = ["10.0.0.0/16"]
    location            = "southcentralus"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_subnet" "test" {
    name                 = "acctsub-%d"
    resource_group_name  = "${azurerm_resource_group.test.name}"
    virtual_network_name = "${azurerm_virtual_network.test.name}"
    address_prefix       = "10.0.2.0/24"
}

resource "azurerm_storage_account" "test" {
    name                = "accsa%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    location            = "southcentralus"
    account_type        = "Standard_LRS"
}

resource "azurerm_storage_container" "test" {
    name                  = "vhds"
    resource_group_name   = "${azurerm_resource_group.test.name}"
    storage_account_name  = "${azurerm_storage_account.test.name}"
    container_access_type = "private"
}

resource "azurerm_virtual_machine_scale_set" "test" {
  	name                = "acctvmss-%d"
  	location            = "southcentralus"
  	resource_group_name = "${azurerm_resource_group.test.name}"
  	upgrade_policy_mode = "Manual"
	overprovision       = false

  	sku {
		name     = "Standard_D1_v2"
    	tier     = "Standard"
    	capacity = 1
	}

  	os_profile {
    	computer_name_prefix = "testvm-%d"
    	admin_username = "myadmin"
    	admin_password = "Passwword1234"
  	}

  	network_profile {
      	name    = "TestNetworkProfile"
      	primary = true
      	ip_configuration {
        	name	  = "TestIPConfiguration"
        	subnet_id = "${azurerm_subnet.test.id}"
      	}
  	}

  	storage_profile_os_disk {
    	name 		   = "os-disk"
    	caching        = "ReadWrite"
    	create_option  = "FromImage"
    	vhd_containers = [ "${azurerm_storage_account.test.primary_blob_endpoint}${azurerm_storage_container.test.name}" ]
  	}

  	storage_profile_image_reference {
    	publisher = "Canonical"
    	offer     = "UbuntuServer"
    	sku       = "16.04-LTS"
    	version   = "latest"
  	}

	extension {
		name                       = "CustomScript"
		publisher                  = "Microsoft.Azure.Extensions"
		type                       = "CustomScript"
		type_handler_version       = "2.0"
		auto_upgrade_minor_version = true
		settings                   = <<SETTINGS
		{
			"commandToExecute": "echo $HOSTNAME"
		}
SETTINGS

		protected_settings         = <<SETTINGS
		{
			"storageAccountName": "${azurerm_storage_account.test.name}",
			"storageAccountKey": "${azurerm_storage_account.test.primary_access_key}"
		}
SETTINGS
	}
}
`

var testAccAzureRMVirtualMachineScaleSetMultipleExtensionsTemplate = `
resource "azurerm_resource_group" "test" {
    name 	 = "acctestrg-%d"
    location = "southcentralus"
}

resource "azurerm_virtual_network" "test" {
    name 		        = "acctvn-%d"
    address_space       = ["10.0.0.0/16"]
    location            = "southcentralus"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_subnet" "test" {
    name                 = "acctsub-%d"
    resource_group_name  = "${azurerm_resource_group.test.name}"
    virtual_network_name = "${azurerm_virtual_network.test.name}"
    address_prefix       = "10.0.2.0/24"
}

resource "azurerm_storage_account" "test" {
    name                = "accsa%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    location            = "southcentralus"
    account_type        = "Standard_LRS"
}

resource "azurerm_storage_container" "test" {
    name                  = "vhds"
    resource_group_name   = "${azurerm_resource_group.test.name}"
    storage_account_name  = "${azurerm_storage_account.test.name}"
    container_access_type = "private"
}

resource "azurerm_virtual_machine_scale_set" "test" {
  	name                = "acctvmss-%d"
  	location            = "southcentralus"
  	resource_group_name = "${azurerm_resource_group.test.name}"
  	upgrade_policy_mode = "Manual"
	overprovision       = false

  	sku {
		name     = "Standard_D1_v2"
    	tier     = "Standard"
    	capacity = 1
	}

  	os_profile {
    	computer_name_prefix = "testvm-%d"
    	admin_username = "myadmin"
    	admin_password = "Passwword1234"
  	}

  	network_profile {
      	name    = "TestNetworkProfile"
      	primary = true
      	ip_configuration {
        	name	  = "TestIPConfiguration"
        	subnet_id = "${azurerm_subnet.test.id}"
      	}
  	}

  	storage_profile_os_disk {
    	name 		   = "os-disk"
    	caching        = "ReadWrite"
    	create_option  = "FromImage"
    	vhd_containers = [ "${azurerm_storage_account.test.primary_blob_endpoint}${azurerm_storage_container.test.name}" ]
  	}

  	storage_profile_image_reference {
    	publisher = "Canonical"
    	offer     = "UbuntuServer"
    	sku       = "16.04-LTS"
    	version   = "latest"
  	}

	extension {
		name                       = "CustomScript"
		publisher                  = "Microsoft.Azure.Extensions"
		type                       = "CustomScript"
		type_handler_version       = "2.0"
		auto_upgrade_minor_version = true
		settings                   = <<SETTINGS
		{
			"commandToExecute": "echo $HOSTNAME"
		}
SETTINGS

		protected_settings         = <<SETTINGS
		{
			"storageAccountName": "${azurerm_storage_account.test.name}",
			"storageAccountKey": "${azurerm_storage_account.test.primary_access_key}"
		}
SETTINGS
	}

	extension {
		name                       = "Docker"
		publisher                  = "Microsoft.Azure.Extensions"
		type                       = "DockerExtension"
		type_handler_version       = "1.0"
		auto_upgrade_minor_version = true
	}
}
`

var testAccAzureRMVirtualMachineScaleSet_osDiskTypeConflict = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US 2"
}

resource "azurerm_virtual_network" "test" {
    name = "acctvn-%d"
    address_space = ["10.0.0.0/16"]
    location = "West US 2"
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
    location = "West US 2"
    resource_group_name = "${azurerm_resource_group.test.name}"

    ip_configuration {
    	name = "testconfiguration1"
    	subnet_id = "${azurerm_subnet.test.id}"
    	private_ip_address_allocation = "dynamic"
    }
}

resource "azurerm_virtual_machine_scale_set" "test" {
  name = "acctvmss-%d"
  location = "West US 2"
  resource_group_name = "${azurerm_resource_group.test.name}"
  upgrade_policy_mode = "Manual"

  sku {
    name = "Standard_D1_v2"
    tier = "Standard"
    capacity = 2
  }

  os_profile {
    computer_name_prefix = "testvm-%d"
    admin_username = "myadmin"
    admin_password = "Passwword1234"
  }

  network_profile {
      name = "TestNetworkProfile-%d"
      primary = true
      ip_configuration {
        name = "TestIPConfiguration"
        subnet_id = "${azurerm_subnet.test.id}"
      }
  }

  storage_profile_os_disk {
	name 		  = ""
    caching       = "ReadWrite"
    create_option = "FromImage"
    managed_disk_type = "Standard_LRS"
    vhd_containers = ["should_cause_conflict"]
  }

  storage_profile_image_reference {
    publisher = "Canonical"
    offer     = "UbuntuServer"
    sku       = "16.04-LTS"
    version   = "latest"
  }
}
`

var testAccAzureRMVirtualMachineScaleSetLoadbalancerTemplateManagedDataDisks = `
resource "azurerm_resource_group" "test" {
    name 	 = "acctestrg-%d"
    location = "southcentralus"
}
resource "azurerm_virtual_network" "test" {
    name 		        = "acctvn-%d"
    address_space       = ["10.0.0.0/16"]
    location            = "southcentralus"
    resource_group_name = "${azurerm_resource_group.test.name}"
}
resource "azurerm_subnet" "test" {
    name                 = "acctsub-%d"
    resource_group_name  = "${azurerm_resource_group.test.name}"
    virtual_network_name = "${azurerm_virtual_network.test.name}"
    address_prefix       = "10.0.2.0/24"
}
resource "azurerm_lb" "test" {
    name                = "acctestlb-%d"
    location            = "southcentralus"
    resource_group_name = "${azurerm_resource_group.test.name}"
    frontend_ip_configuration {
        name                          = "default"
        subnet_id                     = "${azurerm_subnet.test.id}"
        private_ip_address_allocation = "Dynamic"
    }
}
resource "azurerm_lb_backend_address_pool" "test" {
    name                = "test"
    resource_group_name = "${azurerm_resource_group.test.name}"
    loadbalancer_id     = "${azurerm_lb.test.id}"
}
resource "azurerm_virtual_machine_scale_set" "test" {
  	name                = "acctvmss-%d"
  	location            = "southcentralus"
  	resource_group_name = "${azurerm_resource_group.test.name}"
  	upgrade_policy_mode = "Manual"
  	sku {
		name     = "Standard_A0"
    	tier     = "Standard"
    	capacity = 1
	}
  	os_profile {
    	computer_name_prefix = "testvm-%d"
    	admin_username = "myadmin"
    	admin_password = "Passwword1234"
  	}
  	network_profile {
      	name    = "TestNetworkProfile"
      	primary = true
      	ip_configuration {
        	name                                   = "TestIPConfiguration"
        	subnet_id                              = "${azurerm_subnet.test.id}"
			load_balancer_backend_address_pool_ids = [ "${azurerm_lb_backend_address_pool.test.id}" ]
      	}
  	}

  	storage_profile_os_disk {
    	name = ""
    	caching       = "ReadWrite"
    	create_option = "FromImage"
    	managed_disk_type = "Standard_LRS"
  	}
		  
  	storage_profile_data_disk {
		lun 		   = 0
    	caching        = "ReadWrite"
    	create_option  = "Empty"
		disk_size_gb   = 10
	    managed_disk_type = "Standard_LRS"	
  	}

  	storage_profile_image_reference {
    	publisher = "Canonical"
    	offer     = "UbuntuServer"
    	sku       = "16.04.0-LTS"
    	version   = "latest"
  	}
}
`

func testAccAzureRMVirtualMachineScaleSetNonStandardCasing(ri int) string {
	return fmt.Sprintf(`
resource "azurerm_resource_group" "test" {
  name     = "acctestRG-%d"
  location = "West US 2"
}
resource "azurerm_virtual_network" "test" {
  name                = "acctvn-%d"
  address_space       = ["10.0.0.0/16"]
  location            = "West US 2"
  resource_group_name = "${azurerm_resource_group.test.name}"
}
resource "azurerm_subnet" "test" {
  name                 = "acctsub-%d"
  resource_group_name  = "${azurerm_resource_group.test.name}"
  virtual_network_name = "${azurerm_virtual_network.test.name}"
  address_prefix       = "10.0.2.0/24"
}
resource "azurerm_network_interface" "test" {
  name                = "acctni-%d"
  location            = "West US 2"
  resource_group_name = "${azurerm_resource_group.test.name}"
  ip_configuration {
    name                          = "testconfiguration1"
    subnet_id                     = "${azurerm_subnet.test.id}"
    private_ip_address_allocation = "dynamic"
  }
}
resource "azurerm_storage_account" "test" {
  name                = "accsa%d"
  resource_group_name = "${azurerm_resource_group.test.name}"
  location            = "westus2"
  account_type        = "Standard_LRS"
  tags {
    environment = "staging"
  }
}
resource "azurerm_storage_container" "test" {
  name                  = "vhds"
  resource_group_name   = "${azurerm_resource_group.test.name}"
  storage_account_name  = "${azurerm_storage_account.test.name}"
  container_access_type = "private"
}
resource "azurerm_virtual_machine_scale_set" "test" {
  name                = "acctvmss-%d"
  location            = "West US 2"
  resource_group_name = "${azurerm_resource_group.test.name}"
  upgrade_policy_mode = "Manual"
  sku {
    name     = "Standard_A0"
    tier     = "standard"
    capacity = 2
  }
  os_profile {
    computer_name_prefix = "testvm-%d"
    admin_username       = "myadmin"
    admin_password       = "Passwword1234"
  }
  network_profile {
    name    = "TestNetworkProfile-%d"
    primary = true
    ip_configuration {
      name      = "TestIPConfiguration"
      subnet_id = "${azurerm_subnet.test.id}"
    }
  }
  storage_profile_os_disk {
    name           = "osDiskProfile"
    caching        = "ReadWrite"
    create_option  = "FromImage"
    vhd_containers = ["${azurerm_storage_account.test.primary_blob_endpoint}${azurerm_storage_container.test.name}"]
  }
  storage_profile_image_reference {
    publisher = "Canonical"
    offer     = "UbuntuServer"
    sku       = "14.04.2-LTS"
    version   = "latest"
  }
}
`, ri, ri, ri, ri, ri, ri, ri, ri)
}
