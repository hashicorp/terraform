package azurerm

import (
	"fmt"
	"log"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAzureRMManagedImage_standaloneImage(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMManagedImage_standaloneImage, ri, ri)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMManagedImageDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMManagedImageExists("azurerm_managed_image.test", true),
				),
			},
		},
	})
}

func TestAccAzureRMManagedImage_customImageVMFromVHD(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMManagedImage_customImage_fromVHD, ri, ri, ri, ri)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMManagedImageDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMManagedImageExists("azurerm_managed_image.test", true),
				),
			},
		},
	})
}

func TestAccAzureRMManagedImage_customImageVMFromVM(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMManagedImage_customImage_fromVM, ri, ri, ri, ri, ri)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMManagedImageDestroy,
		Steps: []resource.TestStep{
			{
				//need to create a vm and then reference it in the image creation
				Config:             config,
				Destroy:            false,
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureVMExists("azurerm_virtual_machine.testdestination", true),
				),
			},
		},
	})
}

func testCheckAzureRMManagedImageExists(name string, shouldExist bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		log.Printf("[INFO] testing MANAGED IMAGE EXISTS - BEGIN.")

		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		dName := rs.Primary.Attributes["name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for image: %s", dName)
		}

		conn := testAccProvider.Meta().(*ArmClient).imageClient

		resp, err := conn.Get(resourceGroup, dName, "")
		if err != nil {
			return fmt.Errorf("Bad: Get on imageClient: %s", err)
		}

		if resp.StatusCode == http.StatusNotFound && shouldExist {
			return fmt.Errorf("Bad: ManagedImage %q (resource group %q) does not exist", dName, resourceGroup)
		}
		if resp.StatusCode != http.StatusNotFound && !shouldExist {
			return fmt.Errorf("Bad: ManagedImage %q (resource group %q) still exists", dName, resourceGroup)
		}

		return nil
	}
}

func testCheckAzureVMExists(sourceVM string, shouldExist bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		log.Printf("[INFO] testing MANAGED IMAGE VM EXISTS - BEGIN.")

		vmClient := testAccProvider.Meta().(*ArmClient).vmClient
		vmRs, vmOk := s.RootModule().Resources[sourceVM]
		if !vmOk {
			return fmt.Errorf("VM Not found: %s", sourceVM)
		}
		vmName := vmRs.Primary.Attributes["name"]

		resourceGroup, hasResourceGroup := vmRs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for VM: %s", vmName)
		}

		resp, err := vmClient.Get(resourceGroup, vmName, "")
		if err != nil {
			return fmt.Errorf("Bad: Get on vmClient: %s", err)
		}

		if resp.StatusCode == http.StatusNotFound && shouldExist {
			return fmt.Errorf("Bad: VM %q (resource group %q) does not exist", vmName, resourceGroup)
		}
		if resp.StatusCode != http.StatusNotFound && !shouldExist {
			return fmt.Errorf("Bad: VM %q (resource group %q) still exists", vmName, resourceGroup)
		}

		log.Printf("[INFO] testing MANAGED IMAGE VM EXISTS - END.")

		return nil
	}
}

func testCheckAzureRMManagedImageDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*ArmClient).diskClient

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_managed_disk" {
			continue
		}

		name := rs.Primary.Attributes["name"]
		resourceGroup := rs.Primary.Attributes["resource_group_name"]

		resp, err := conn.Get(resourceGroup, name)

		if err != nil {
			return nil
		}

		if resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("Managed Disk still exists: \n%#v", resp.Properties)
		}
	}

	return nil
}

var testAccAzureRMManagedImage_standaloneImage = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West Central US"
}

resource "azurerm_managed_image" "test" {
    name = "accteste-%d"
    location = "West Central US"
    resource_group_name = "${azurerm_resource_group.test.name}"
	os_disk {
		os_disk_ostype = "Linux"
		os_disk_osstate = "Generalized"
		os_disk_blob_uri = "https://terraformdev.blob.core.windows.net/packerimages/ubuntu_plain.vhd"
 	    os_disk_size_gb = 30
	}

    tags {
        environment = "acctest"
        cost-center = "ops"
    }
}`

var testAccAzureRMManagedImage_customImage_fromVHD = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West Central US"
}

resource "azurerm_virtual_network" "test" {
    name = "acctvn-%d"
    address_space = ["10.0.0.0/16"]
    location = "West Central US"
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
    location = "West Central US"
    resource_group_name = "${azurerm_resource_group.test.name}"

    ip_configuration {
    	name = "testconfiguration1"
    	subnet_id = "${azurerm_subnet.test.id}"
    	private_ip_address_allocation = "dynamic"
    }
}

resource "azurerm_managed_image" "test" {
    name = "accteste"
    location = "West Central US"
    resource_group_name = "${azurerm_resource_group.test.name}"
	os_disk {
		os_disk_ostype = "Linux"
		os_disk_osstate = "Generalized"
		os_disk_blob_uri = "https://terraformdev.blob.core.windows.net/packerimages/ubuntu_plain.vhd"
 	    os_disk_size_gb = 30
	}

    tags {
        environment = "acctest"
        cost-center = "ops"
    }
}

resource "azurerm_virtual_machine" "test" {
    name = "acctvm"
    location = "West Central US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    network_interface_ids = ["${azurerm_network_interface.test.id}"]
    vm_size = "Standard_D1_v2"

    storage_image_reference {
		image_id = "${azurerm_managed_image.test.id}"
    }

    storage_os_disk {
        name = "myosdisk1"
        caching = "ReadWrite"
        create_option = "FromImage"
    }
	
    os_profile {
		computer_name = "mdcustomimagetest"
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

var testAccAzureRMManagedImage_customImage_fromVM = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West Central US"
}

resource "azurerm_managed_image" "testdestination" {
    name = "acctestdest-%d"
    location = "West Central US"
    resource_group_name = "${azurerm_resource_group.test.name}"
source_virtual_machine_id = "/subscriptions/{subscription_id}/resourceGroups/{resource_group}/providers/Microsoft.Compute/virtualMachines/{vm_name}"    
	tags {
        environment = "acctest"
        cost-center = "ops"
    }
}

resource "azurerm_virtual_network" "test" {
    name = "acctvn-%d"
    address_space = ["10.0.0.0/16"]
    location = "West Central US"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_subnet" "test" {
    name = "acctsub-%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    virtual_network_name = "${azurerm_virtual_network.test.name}"
    address_prefix = "10.0.2.0/24"
}

resource "azurerm_network_interface" "testdestination" {
    name = "acctnicdest-%d"
    location = "West Central US"
    resource_group_name = "${azurerm_resource_group.test.name}"

    ip_configuration {
    	name = "testconfiguration2"
    	subnet_id = "${azurerm_subnet.test.id}"
    	private_ip_address_allocation = "dynamic"
    }
}

resource "azurerm_virtual_machine" "testdestination" {
    name = "acctvmdest"
    location = "West Central US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    network_interface_ids = ["${azurerm_network_interface.testdestination.id}"]
    vm_size = "Standard_D1_v2"

    storage_image_reference {
		image_id = "${azurerm_managed_image.testdestination.id}"
    }

    storage_os_disk {
        name = "myosdisk1"
        caching = "ReadWrite"
        create_option = "FromImage"
    }
	
    os_profile {
		computer_name = "mdcustomimagetest"
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
