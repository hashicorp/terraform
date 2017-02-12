package azurerm

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAzureRMVirtualMachineScaleSet_basicLinux(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMVirtualMachineScaleSet_basicLinux, ri, ri, ri, ri, ri, ri, ri, ri)
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
	config := fmt.Sprintf(testAccAzureRMVirtualMachineScaleSet_basicLinux, ri, ri, ri, ri, ri, ri, ri, ri)
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

func testCheckAzureRMVirtualMachineScaleSetExists(name string) resource.TestCheckFunc {
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

		return nil
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

		_, err := conn.Delete(resourceGroup, name, make(chan struct{}))
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

		if *resp.Overprovision {
			return fmt.Errorf("Bad: Overprovision should have been false for scale set %v", name)
		}

		return nil
	}
}

var testAccAzureRMVirtualMachineScaleSet_basicLinux = `
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

resource "azurerm_virtual_machine_scale_set" "test" {
  name = "acctvmss-%d"
  location = "West US"
  resource_group_name = "${azurerm_resource_group.test.name}"
  upgrade_policy_mode = "Manual"

  sku {
    name = "Standard_A0"
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
    sku       = "14.04.2-LTS"
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
    	name 		   = "os-disk"
    	caching        = "ReadWrite"
    	create_option  = "FromImage"
    	vhd_containers = [ "${azurerm_storage_account.test.primary_blob_endpoint}${azurerm_storage_container.test.name}" ]
  	}

  	storage_profile_image_reference {
    	publisher = "Canonical"
    	offer     = "UbuntuServer"
    	sku       = "14.04.2-LTS"
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
    	sku       = "14.04.2-LTS"
    	version   = "latest"
  	}
}
`
