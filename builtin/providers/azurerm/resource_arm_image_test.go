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

func TestAccAzureRMImage_standaloneImage(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMImage_standaloneImage, ri)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMImageDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMImageExists("azurerm_image.test", true),
				),
			},
		},
	})
}

func TestAccAzureRMImage_customImageVMFromVHD(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMImage_customImage_fromVHD, ri)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMImageDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMImageExists("azurerm_image.test", true),
				),
			},
		},
	})
}

func TestAccAzureRMImage_customImageVMFromVM(t *testing.T) {
	ri := acctest.RandInt()
	preConfig := fmt.Sprintf(testAccAzureRMImage_customImage_fromVM_sourceVM, ri)
	postConfig := fmt.Sprintf(testAccAzureRMImage_customImage_fromVM_destinationVM, ri)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMImageDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				//need to create a vm and then reference it in the image creation
				Config:  preConfig,
				Destroy: false,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureVMExists("azurerm_virtual_machine.testsource", true),
					testGeneralizeVMImage(fmt.Sprintf("acctestRG-%[1]d", ri), "testsource"),
				),
			},
			resource.TestStep{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureVMExists("azurerm_virtual_machine.testdestination", true),
				),
			},
		},
	})
}

func testGeneralizeVMImage(groupName string, vmName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		vmClient := testAccProvider.Meta().(*ArmClient).vmClient

		_, powerOffErr := vmClient.PowerOff(groupName, vmName, nil)
		if powerOffErr != nil {
			return fmt.Errorf("Bad: Powering off error %s", powerOffErr)
		}

		_, generalizeErr := vmClient.Generalize(groupName, vmName)
		if generalizeErr != nil {
			return fmt.Errorf("Bad: Generalizing off error %s", generalizeErr)
		}

		return nil
	}
}

func testCheckAzureRMImageExists(name string, shouldExist bool) resource.TestCheckFunc {
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
			return fmt.Errorf("Bad: Image %q (resource group %q) does not exist", dName, resourceGroup)
		}
		if resp.StatusCode != http.StatusNotFound && !shouldExist {
			return fmt.Errorf("Bad: Image %q (resource group %q) still exists", dName, resourceGroup)
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

func testCheckAzureRMImageDestroy(s *terraform.State) error {
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

var testAccAzureRMImage_standaloneImage = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%[1]d"
    location = "West Central US"
}

resource "azurerm_image" "test" {
    name = "accteste-%[1]d"
    location = "West Central US"
    resource_group_name = "${azurerm_resource_group.test.name}"
	os_disk_os_type = "Linux"
	os_disk_os_state = "Generalized"
	os_disk_blob_uri = "https://terraformdev.blob.core.windows.net/packerimages/ubuntu_plain.vhd"
	os_disk_size_gb = 30
	os_disk_caching = "None"

    tags {
        environment = "acctest"
        cost-center = "ops"
    }
}`

var testAccAzureRMImage_customImage_fromVHD = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%[1]d"
    location = "West Central US"
}

resource "azurerm_virtual_network" "test" {
    name = "acctvn-%[1]d"
    address_space = ["10.0.0.0/16"]
    location = "West Central US"
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
    location = "West Central US"
    resource_group_name = "${azurerm_resource_group.test.name}"

    ip_configuration {
    	name = "testconfiguration1"
    	subnet_id = "${azurerm_subnet.test.id}"
    	private_ip_address_allocation = "dynamic"
    }
}

resource "azurerm_image" "test" {
    name = "accteste"
    location = "West Central US"
    resource_group_name = "${azurerm_resource_group.test.name}"
	os_disk_os_type = "Linux"
	os_disk_os_state = "Generalized"
	os_disk_blob_uri = "https://terraformdev.blob.core.windows.net/packerimages/ubuntu_plain.vhd"
	os_disk_size_gb = 30
	os_disk_caching = "None"		 

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
		image_id = "${azurerm_image.test.id}"
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

var testAccAzureRMImage_customImage_fromVM_sourceVM = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%[1]d"
    location = "West Central US"
}

resource "azurerm_virtual_network" "test" {
    name = "acctvn-%[1]d"
    address_space = ["10.0.0.0/16"]
    location = "West Central US"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_subnet" "test" {
    name = "acctsub-%[1]d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    virtual_network_name = "${azurerm_virtual_network.test.name}"
    address_prefix = "10.0.2.0/24"
}

resource "azurerm_network_interface" "testsource" {
    name = "acctnicsource-%[1]d"
    location = "West Central US"
    resource_group_name = "${azurerm_resource_group.test.name}"

    ip_configuration {
    	name = "testconfigurationsource"
    	subnet_id = "${azurerm_subnet.test.id}"
    	private_ip_address_allocation = "dynamic"
    }
}

resource "azurerm_virtual_machine" "testsource" {
    name = "testsource"
    location = "West Central US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    network_interface_ids = ["${azurerm_network_interface.testsource.id}"]
    vm_size = "Standard_D1_v2"

    storage_image_reference {
		publisher = "Canonical"
		offer = "UbuntuServer"
		sku = "16.04-LTS"
		version = "latest"
    }

    storage_os_disk {
        name = "myosdisk1"
        caching = "ReadWrite"
        create_option = "FromImage"
    }
	
    os_profile {
		computer_name = "mdimagetestsource"
		admin_username = "testadmin"
		admin_password = "Password1234!"
    }

    os_profile_linux_config {
		disable_password_authentication = false
    }

    tags {
    	environment = "Dev"
    	cost-center = "Ops"
    }
}
`

var testAccAzureRMImage_customImage_fromVM_destinationVM = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%[1]d"
    location = "West Central US"
}

resource "azurerm_virtual_network" "test" {
    name = "acctvn-%[1]d"
    address_space = ["10.0.0.0/16"]
    location = "West Central US"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_subnet" "test" {
    name = "acctsub-%[1]d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    virtual_network_name = "${azurerm_virtual_network.test.name}"
    address_prefix = "10.0.2.0/24"
}

resource "azurerm_network_interface" "testsource" {
    name = "acctnicsource-%[1]d"
    location = "West Central US"
    resource_group_name = "${azurerm_resource_group.test.name}"

    ip_configuration {
    	name = "testconfigurationsource"
    	subnet_id = "${azurerm_subnet.test.id}"
    	private_ip_address_allocation = "dynamic"
    }
}

resource "azurerm_virtual_machine" "testsource" {
    name = "testsource"
    location = "West Central US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    network_interface_ids = ["${azurerm_network_interface.testsource.id}"]
    vm_size = "Standard_D1_v2"

    storage_image_reference {
		publisher = "Canonical"
		offer = "UbuntuServer"
		sku = "16.04-LTS"
		version = "latest"
    }

    storage_os_disk {
        name = "myosdisk1"
        caching = "ReadWrite"
        create_option = "FromImage"
    }
	
    os_profile {
		computer_name = "mdimagetestsource"
		admin_username = "testadmin"
		admin_password = "Password1234!"
    }

    os_profile_linux_config {
		disable_password_authentication = false
    }

    tags {
    	environment = "Dev"
    	cost-center = "Ops"
    }
}

resource "azurerm_image" "testdestination" {
    name = "acctestdest-%[1]d"
    location = "West Central US"
    resource_group_name = "${azurerm_resource_group.test.name}"
	source_virtual_machine_id = "${azurerm_virtual_machine.testsource.id}"    
	tags {
        environment = "acctest"
        cost-center = "ops"
    }
}

resource "azurerm_network_interface" "testdestination" {
    name = "acctnicdest-%[1]d"
    location = "West Central US"
    resource_group_name = "${azurerm_resource_group.test.name}"

    ip_configuration {
    	name = "testconfiguration2"
    	subnet_id = "${azurerm_subnet.test.id}"
    	private_ip_address_allocation = "dynamic"
    }
}

resource "azurerm_virtual_machine" "testdestination" {
    name = "testdestination"
    location = "West Central US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    network_interface_ids = ["${azurerm_network_interface.testdestination.id}"]
    vm_size = "Standard_D1_v2"

    storage_image_reference {
		image_id = "${azurerm_image.testdestination.id}"
    }

    storage_os_disk {
        name = "myosdisk1"
        caching = "ReadWrite"
        create_option = "FromImage"
    }
	
    os_profile {
		computer_name = "mdimagetestdest"
		admin_username = "testadmin"
		admin_password = "Password1234!"
    }

    os_profile_linux_config {
		disable_password_authentication = false
    }

    tags {
    	environment = "Dev"
    	cost-center = "Ops"
    }
}
`
