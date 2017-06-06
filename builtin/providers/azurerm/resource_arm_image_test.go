package azurerm

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAzureRMImage_standaloneImage(t *testing.T) {
	ri := acctest.RandInt()
	userName := "testadmin"
	password := "Password1234!"
	hostName := "tftestcustomimagesrc"
	dnsName := fmt.Sprintf("%[1]s.westcentralus.cloudapp.azure.com", hostName)
	sshPort := "22"
	preConfig := fmt.Sprintf(testAccAzureRMImage_standaloneImage_setup, ri, userName, password, hostName)
	postConfig := fmt.Sprintf(testAccAzureRMImage_standaloneImage_provision, ri, userName, password, hostName)

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
					testGeneralizeVMImage(fmt.Sprintf("acctestRG-%[1]d", ri), "testsource",
						userName, password, dnsName, sshPort),
				),
			},
			resource.TestStep{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMImageExists("azurerm_image.test", true),
				),
			},
		},
	})
}

func TestAccAzureRMImage_customImageVMFromVHD(t *testing.T) {
	ri := acctest.RandInt()
	userName := "testadmin"
	password := "Password1234!"
	hostName := "tftestcustomimagesrc"
	dnsName := fmt.Sprintf("%[1]s.westcentralus.cloudapp.azure.com", hostName)
	sshPort := "22"
	preConfig := fmt.Sprintf(testAccAzureRMImage_customImage_fromVHD_setup, ri, userName, password, hostName)
	postConfig := fmt.Sprintf(testAccAzureRMImage_customImage_fromVHD_provision, ri, userName, password, hostName)

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
					testGeneralizeVMImage(fmt.Sprintf("acctestRG-%[1]d", ri), "testsource",
						userName, password, dnsName, sshPort),
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

func TestAccAzureRMImage_customImageVMFromVM(t *testing.T) {
	ri := acctest.RandInt()
	userName := "testadmin"
	password := "Password1234!"
	hostName := "tftestcustomimagesrc"
	dnsName := fmt.Sprintf("%[1]s.westcentralus.cloudapp.azure.com", hostName)
	sshPort := "22"
	preConfig := fmt.Sprintf(testAccAzureRMImage_customImage_fromVM_sourceVM, ri, userName, password, hostName)
	postConfig := fmt.Sprintf(testAccAzureRMImage_customImage_fromVM_destinationVM, ri, userName, password, hostName)

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
					testGeneralizeVMImage(fmt.Sprintf("acctestRG-%[1]d", ri), "testsource",
						userName, password, dnsName, sshPort),
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

func testGeneralizeVMImage(groupName string, vmName string, userName string, password string, hostName string, port string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		vmClient := testAccProvider.Meta().(*ArmClient).vmClient

		deprovisionErr := deprovisionVM(userName, password, hostName, port)
		if deprovisionErr != nil {
			return fmt.Errorf("Bad: Deprovisioning error %s", deprovisionErr)
		}

		_, deallocateErr := vmClient.Deallocate(groupName, vmName, nil)
		err := <-deallocateErr
		if err != nil {
			return fmt.Errorf("Bad: Deallocating error %s", err)
		}

		_, generalizeErr := vmClient.Generalize(groupName, vmName)
		if generalizeErr != nil {
			return fmt.Errorf("Bad: Generalizing error %s", generalizeErr)
		}

		return nil
	}
}

func deprovisionVM(userName string, password string, hostName string, port string) error {
	//SSH into the machine and execute a waagent deprovisioning command
	var b bytes.Buffer
	cmd := "sudo waagent -verbose -deprovision+user -force"

	config := &ssh.ClientConfig{
		User: userName,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		//line below for the newer versions of SSH Client
		//HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	log.Printf("[INFO] Connecting to %s:%v remote server...", hostName, port)

	hostAddress := strings.Join([]string{hostName, port}, ":")
	client, err := ssh.Dial("tcp", hostAddress, config)
	if err != nil {
		return fmt.Errorf("Bad: deprovisioning error %s", err.Error())
	}

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("Bad: deprovisioning error, failure creating session %s", err.Error())
	}
	defer session.Close()

	session.Stdout = &b
	if err := session.Run(cmd); err != nil {
		return fmt.Errorf("Bad: deprovisioning error, failure running command %s", err.Error())
	}

	return nil
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
		if rs.Type != "azurerm_image" {
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

var testAccAzureRMImage_standaloneImage_setup = `
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

resource "azurerm_public_ip" "test" {
  name                         = "acctpip-%[1]d"
  location                     = "West Central US"
  resource_group_name          = "${azurerm_resource_group.test.name}"
  public_ip_address_allocation = "Dynamic"
  domain_name_label            = "%[4]s"
}

resource "azurerm_network_interface" "testsource" {
    name = "acctnicsource-%[1]d"
    location = "West Central US"
    resource_group_name = "${azurerm_resource_group.test.name}"

    ip_configuration {
    	name = "testconfigurationsource"
    	subnet_id = "${azurerm_subnet.test.id}"
    	private_ip_address_allocation = "dynamic"
	    public_ip_address_id          = "${azurerm_public_ip.test.id}"
    }
}

resource "azurerm_storage_account" "test" {
    name = "accsa%[1]d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    location = "West Central US"
    account_type = "Standard_LRS"

    tags {
        environment = "Dev"
    }
}

resource "azurerm_storage_container" "test" {
    name = "vhds"
    resource_group_name = "${azurerm_resource_group.test.name}"
    storage_account_name = "${azurerm_storage_account.test.name}"
    container_access_type = "blob"
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
        vhd_uri = "${azurerm_storage_account.test.primary_blob_endpoint}${azurerm_storage_container.test.name}/myosdisk1.vhd"
        caching = "ReadWrite"
        create_option = "FromImage"
        disk_size_gb = "30"
    }
	
    os_profile {
		computer_name = "mdimagetestsource"
		admin_username = "%[2]s"
		admin_password = "%[3]s"
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

var testAccAzureRMImage_standaloneImage_provision = `
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

resource "azurerm_public_ip" "test" {
	name                         = "acctpip-%[1]d"
	location                     = "West Central US"
	resource_group_name          = "${azurerm_resource_group.test.name}"
	public_ip_address_allocation = "Dynamic"
	domain_name_label            = "%[4]s"
}

resource "azurerm_network_interface" "testsource" {
	name = "acctnicsource-%[1]d"
	location = "West Central US"
	resource_group_name = "${azurerm_resource_group.test.name}"

	ip_configuration {
		name = "testconfigurationsource"
		subnet_id = "${azurerm_subnet.test.id}"
		private_ip_address_allocation = "dynamic"
		public_ip_address_id          = "${azurerm_public_ip.test.id}"
	}
}

resource "azurerm_storage_account" "test" {
	name = "accsa%[1]d"
	resource_group_name = "${azurerm_resource_group.test.name}"
	location = "West Central US"
	account_type = "Standard_LRS"

	tags {
		environment = "Dev"
	}
}

resource "azurerm_storage_container" "test" {
	name = "vhds"
	resource_group_name = "${azurerm_resource_group.test.name}"
	storage_account_name = "${azurerm_storage_account.test.name}"
	container_access_type = "blob"
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
		vhd_uri = "${azurerm_storage_account.test.primary_blob_endpoint}${azurerm_storage_container.test.name}/myosdisk1.vhd"
		caching = "ReadWrite"
		create_option = "FromImage"
		disk_size_gb = "45"
	}
	
	os_profile {
		computer_name = "mdimagetestsource"
		admin_username = "%[2]s"
		admin_password = "%[3]s"
	}

	os_profile_linux_config {
		disable_password_authentication = false
	}

	tags {
		environment = "Dev"
		cost-center = "Ops"
	}
}

resource "azurerm_image" "test" {
	name = "accteste"
	location = "West Central US"
	resource_group_name = "${azurerm_resource_group.test.name}"
	os_disk_os_type = "Linux"
	os_disk_os_state = "Generalized"
	os_disk_blob_uri = "${azurerm_storage_account.test.primary_blob_endpoint}${azurerm_storage_container.test.name}/myosdisk1.vhd"
	os_disk_size_gb = 30
	os_disk_caching = "None"		 

	tags {
		environment = "Dev"
		cost-center = "Ops"
	}
}
`

var testAccAzureRMImage_customImage_fromVHD_setup = `
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

resource "azurerm_public_ip" "test" {
  name                         = "acctpip-%[1]d"
  location                     = "West Central US"
  resource_group_name          = "${azurerm_resource_group.test.name}"
  public_ip_address_allocation = "Dynamic"
  domain_name_label            = "%[4]s"
}

resource "azurerm_network_interface" "testsource" {
    name = "acctnicsource-%[1]d"
    location = "West Central US"
    resource_group_name = "${azurerm_resource_group.test.name}"

    ip_configuration {
    	name = "testconfigurationsource"
    	subnet_id = "${azurerm_subnet.test.id}"
    	private_ip_address_allocation = "dynamic"
	    public_ip_address_id          = "${azurerm_public_ip.test.id}"
    }
}

resource "azurerm_storage_account" "test" {
    name = "accsa%[1]d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    location = "West Central US"
    account_type = "Standard_LRS"

    tags {
        environment = "Dev"
    }
}

resource "azurerm_storage_container" "test" {
    name = "vhds"
    resource_group_name = "${azurerm_resource_group.test.name}"
    storage_account_name = "${azurerm_storage_account.test.name}"
    container_access_type = "blob"
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
        vhd_uri = "${azurerm_storage_account.test.primary_blob_endpoint}${azurerm_storage_container.test.name}/myosdisk1.vhd"
        caching = "ReadWrite"
        create_option = "FromImage"
        disk_size_gb = "30"
    }
	
    os_profile {
		computer_name = "mdimagetestsource"
		admin_username = "%[2]s"
		admin_password = "%[3]s"
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

var testAccAzureRMImage_customImage_fromVHD_provision = `
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

resource "azurerm_public_ip" "test" {
	name                         = "acctpip-%[1]d"
	location                     = "West Central US"
	resource_group_name          = "${azurerm_resource_group.test.name}"
	public_ip_address_allocation = "Dynamic"
	domain_name_label            = "%[4]s"
}

resource "azurerm_network_interface" "testsource" {
	name = "acctnicsource-%[1]d"
	location = "West Central US"
	resource_group_name = "${azurerm_resource_group.test.name}"

	ip_configuration {
		name = "testconfigurationsource"
		subnet_id = "${azurerm_subnet.test.id}"
		private_ip_address_allocation = "dynamic"
		public_ip_address_id          = "${azurerm_public_ip.test.id}"
	}
}

resource "azurerm_storage_account" "test" {
	name = "accsa%[1]d"
	resource_group_name = "${azurerm_resource_group.test.name}"
	location = "West Central US"
	account_type = "Standard_LRS"

	tags {
		environment = "Dev"
	}
}

resource "azurerm_storage_container" "test" {
	name = "vhds"
	resource_group_name = "${azurerm_resource_group.test.name}"
	storage_account_name = "${azurerm_storage_account.test.name}"
	container_access_type = "blob"
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
		vhd_uri = "${azurerm_storage_account.test.primary_blob_endpoint}${azurerm_storage_container.test.name}/myosdisk1.vhd"
		caching = "ReadWrite"
		create_option = "FromImage"
		disk_size_gb = "45"
	}
	
	os_profile {
		computer_name = "mdimagetestsource"
		admin_username = "%[2]s"
		admin_password = "%[3]s"
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
	name = "accteste"
	location = "West Central US"
	resource_group_name = "${azurerm_resource_group.test.name}"
	os_disk_os_type = "Linux"
	os_disk_os_state = "Generalized"
	os_disk_blob_uri = "${azurerm_storage_account.test.primary_blob_endpoint}${azurerm_storage_container.test.name}/myosdisk1.vhd"
	os_disk_size_gb = 30
	os_disk_caching = "None"		 

	tags {
		environment = "Dev"
		cost-center = "Ops"
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
	name = "acctvm"
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
		computer_name = "mdimagetestsource"
		admin_username = "%[2]s"
		admin_password = "%[3]s"
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

resource "azurerm_public_ip" "test" {
  name                         = "acctpip-%[1]d"
  location                     = "West Central US"
  resource_group_name          = "${azurerm_resource_group.test.name}"
  public_ip_address_allocation = "Dynamic"
  domain_name_label            = "%[4]s"
}

resource "azurerm_network_interface" "testsource" {
    name = "acctnicsource-%[1]d"
    location = "West Central US"
    resource_group_name = "${azurerm_resource_group.test.name}"

    ip_configuration {
    	name = "testconfigurationsource"
    	subnet_id = "${azurerm_subnet.test.id}"
    	private_ip_address_allocation = "dynamic"
	    public_ip_address_id          = "${azurerm_public_ip.test.id}"
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
		admin_username = "%[2]s"
		admin_password = "%[3]s"
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

resource "azurerm_public_ip" "test" {
  name                         = "acctpip-%[1]d"
  location                     = "West Central US"
  resource_group_name          = "${azurerm_resource_group.test.name}"
  public_ip_address_allocation = "Dynamic"
  domain_name_label            = "%[4]s"
}

resource "azurerm_network_interface" "testsource" {
    name = "acctnicsource-%[1]d"
    location = "West Central US"
    resource_group_name = "${azurerm_resource_group.test.name}"

    ip_configuration {
    	name = "testconfigurationsource"
    	subnet_id = "${azurerm_subnet.test.id}"
    	private_ip_address_allocation = "dynamic"
	    public_ip_address_id          = "${azurerm_public_ip.test.id}"		
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
		admin_username = "%[2]s"
		admin_password = "%[3]s"
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
        name = "myosdisk2"
        caching = "ReadWrite"
        create_option = "FromImage"
    }
	
    os_profile {
		computer_name = "mdimagetestdest"
		admin_username = "%[2]s"
		admin_password = "%[3]s"
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
