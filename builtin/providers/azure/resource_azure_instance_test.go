package azure

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/management"
	"github.com/Azure/azure-sdk-for-go/management/virtualmachine"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

var randInt = rand.New(rand.NewSource(time.Now().UnixNano())).Int()
var instanceName = fmt.Sprintf("terraform-test-%d", randInt)

func TestAccAzureInstance_basic(t *testing.T) {
	var dpmt virtualmachine.DeploymentResponse

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAzureInstanceDestroyed(""),
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAzureInstance_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAzureInstanceExists(
						"azure_instance.foo", "", &dpmt),
					testAccCheckAzureInstanceBasicAttributes(&dpmt),
					resource.TestCheckResourceAttr(
						"azure_instance.foo", "name", instanceName),
					resource.TestCheckResourceAttr(
						"azure_instance.foo", "hosted_service_name", instanceName),
					resource.TestCheckResourceAttr(
						"azure_instance.foo", "location", "West US"),
					resource.TestCheckResourceAttr(
						"azure_instance.foo", "endpoint.2462817782.public_port", "22"),
				),
			},
		},
	})
}

func TestAccAzureInstance_separateHostedService(t *testing.T) {
	var dpmt virtualmachine.DeploymentResponse

	hostedServiceName := fmt.Sprintf("terraform-testing-service%d", acctest.RandInt())

	config := fmt.Sprintf(testAccAzureInstance_separateHostedService, hostedServiceName, instanceName, testAccStorageServiceName)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAzureInstanceDestroyed(hostedServiceName),
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAzureInstanceExists(
						"azure_instance.foo", hostedServiceName, &dpmt),
					testAccCheckAzureInstanceBasicAttributes(&dpmt),
					resource.TestCheckResourceAttr(
						"azure_instance.foo", "name", instanceName),
					resource.TestCheckResourceAttr(
						"azure_instance.foo", "hosted_service_name", hostedServiceName),
					resource.TestCheckResourceAttr(
						"azure_instance.foo", "location", "West US"),
					resource.TestCheckResourceAttr(
						"azure_instance.foo", "endpoint.2462817782.public_port", "22"),
				),
			},
		},
	})
}

func TestAccAzureInstance_advanced(t *testing.T) {
	var dpmt virtualmachine.DeploymentResponse

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAzureInstanceDestroyed(""),
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAzureInstance_advanced,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAzureInstanceExists(
						"azure_instance.foo", "", &dpmt),
					testAccCheckAzureInstanceAdvancedAttributes(&dpmt),
					resource.TestCheckResourceAttr(
						"azure_instance.foo", "name", "terraform-test1"),
					resource.TestCheckResourceAttr(
						"azure_instance.foo", "hosted_service_name", "terraform-test1"),
					resource.TestCheckResourceAttr(
						"azure_instance.foo", "size", "Basic_A1"),
					resource.TestCheckResourceAttr(
						"azure_instance.foo", "subnet", "subnet1"),
					resource.TestCheckResourceAttr(
						"azure_instance.foo", "virtual_network", "terraform-vnet-advanced-test"),
					resource.TestCheckResourceAttr(
						"azure_instance.foo", "security_group", "terraform-security-group1"),
					resource.TestCheckResourceAttr(
						"azure_instance.foo", "endpoint.1814039778.public_port", "3389"),
				),
			},
		},
	})
}

func TestAccAzureInstance_update(t *testing.T) {
	var dpmt virtualmachine.DeploymentResponse

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAzureInstanceDestroyed(""),
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAzureInstance_advanced,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAzureInstanceExists(
						"azure_instance.foo", "", &dpmt),
					testAccCheckAzureInstanceAdvancedAttributes(&dpmt),
					resource.TestCheckResourceAttr(
						"azure_instance.foo", "name", "terraform-test1"),
					resource.TestCheckResourceAttr(
						"azure_instance.foo", "hosted_service_name", "terraform-test1"),
					resource.TestCheckResourceAttr(
						"azure_instance.foo", "size", "Basic_A1"),
					resource.TestCheckResourceAttr(
						"azure_instance.foo", "subnet", "subnet1"),
					resource.TestCheckResourceAttr(
						"azure_instance.foo", "virtual_network", "terraform-vnet-advanced-test"),
					resource.TestCheckResourceAttr(
						"azure_instance.foo", "security_group", "terraform-security-group1"),
					resource.TestCheckResourceAttr(
						"azure_instance.foo", "endpoint.1814039778.public_port", "3389"),
				),
			},

			resource.TestStep{
				Config: testAccAzureInstance_update,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAzureInstanceExists(
						"azure_instance.foo", "", &dpmt),
					testAccCheckAzureInstanceUpdatedAttributes(&dpmt),
					resource.TestCheckResourceAttr(
						"azure_instance.foo", "size", "Basic_A2"),
					resource.TestCheckResourceAttr(
						"azure_instance.foo", "security_group", "terraform-security-update-group2"),
					resource.TestCheckResourceAttr(
						"azure_instance.foo", "endpoint.1814039778.public_port", "3389"),
					resource.TestCheckResourceAttr(
						"azure_instance.foo", "endpoint.3713350066.public_port", "5985"),
				),
			},
		},
	})
}

func testAccCheckAzureInstanceExists(
	n string,
	hostedServiceName string,
	dpmt *virtualmachine.DeploymentResponse) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No instance ID is set")
		}

		// if not hosted service was provided; it means that we expect it
		// to be identical with the name of the instance; which is in the ID.
		var serviceName string
		if hostedServiceName == "" {
			serviceName = rs.Primary.ID
		} else {
			serviceName = hostedServiceName
		}

		vmClient := testAccProvider.Meta().(*Client).vmClient
		vm, err := vmClient.GetDeployment(serviceName, rs.Primary.ID)
		if err != nil {
			return err
		}

		if vm.Name != rs.Primary.ID {
			return fmt.Errorf("Instance not found")
		}

		*dpmt = vm

		return nil
	}
}

func testAccCheckAzureInstanceBasicAttributes(
	dpmt *virtualmachine.DeploymentResponse) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if dpmt.Name != instanceName {
			return fmt.Errorf("Bad name: %s", dpmt.Name)
		}

		if len(dpmt.RoleList) != 1 {
			return fmt.Errorf(
				"Instance %s has an unexpected number of roles: %d", dpmt.Name, len(dpmt.RoleList))
		}

		if dpmt.RoleList[0].RoleSize != "Basic_A1" {
			return fmt.Errorf("Bad size: %s", dpmt.RoleList[0].RoleSize)
		}

		return nil
	}
}

func testAccCheckAzureInstanceAdvancedAttributes(
	dpmt *virtualmachine.DeploymentResponse) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if dpmt.Name != "terraform-test1" {
			return fmt.Errorf("Bad name: %s", dpmt.Name)
		}

		if dpmt.VirtualNetworkName != "terraform-vnet-advanced-test" {
			return fmt.Errorf("Bad virtual network: %s", dpmt.VirtualNetworkName)
		}

		if len(dpmt.RoleList) != 1 {
			return fmt.Errorf(
				"Instance %s has an unexpected number of roles: %d", dpmt.Name, len(dpmt.RoleList))
		}

		if dpmt.RoleList[0].RoleSize != "Basic_A1" {
			return fmt.Errorf("Bad size: %s", dpmt.RoleList[0].RoleSize)
		}

		for _, c := range dpmt.RoleList[0].ConfigurationSets {
			if c.ConfigurationSetType == virtualmachine.ConfigurationSetTypeNetwork {
				if len(c.InputEndpoints) != 1 {
					return fmt.Errorf(
						"Instance %s has an unexpected number of endpoints %d",
						dpmt.Name, len(c.InputEndpoints))
				}

				if c.InputEndpoints[0].Name != "RDP" {
					return fmt.Errorf("Bad endpoint name: %s", c.InputEndpoints[0].Name)
				}

				if c.InputEndpoints[0].Port != 3389 {
					return fmt.Errorf("Bad endpoint port: %d", c.InputEndpoints[0].Port)
				}

				if len(c.SubnetNames) != 1 {
					return fmt.Errorf(
						"Instance %s has an unexpected number of associated subnets %d",
						dpmt.Name, len(c.SubnetNames))
				}

				if c.SubnetNames[0] != "subnet1" {
					return fmt.Errorf("Bad subnet: %s", c.SubnetNames[0])
				}

				if c.NetworkSecurityGroup != "terraform-security-group1" {
					return fmt.Errorf("Bad security group: %s", c.NetworkSecurityGroup)
				}
			}
		}

		return nil
	}
}

func testAccCheckAzureInstanceAdvancedUpdatedAttributes(
	dpmt *virtualmachine.DeploymentResponse) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if dpmt.Name != "terraform-test1" {
			return fmt.Errorf("Bad name: %s", dpmt.Name)
		}

		if dpmt.VirtualNetworkName != "terraform-vnet-update-test" {
			return fmt.Errorf("Bad virtual network: %s", dpmt.VirtualNetworkName)
		}

		if len(dpmt.RoleList) != 1 {
			return fmt.Errorf(
				"Instance %s has an unexpected number of roles: %d", dpmt.Name, len(dpmt.RoleList))
		}

		if dpmt.RoleList[0].RoleSize != "Basic_A1" {
			return fmt.Errorf("Bad size: %s", dpmt.RoleList[0].RoleSize)
		}

		for _, c := range dpmt.RoleList[0].ConfigurationSets {
			if c.ConfigurationSetType == virtualmachine.ConfigurationSetTypeNetwork {
				if len(c.InputEndpoints) != 1 {
					return fmt.Errorf(
						"Instance %s has an unexpected number of endpoints %d",
						dpmt.Name, len(c.InputEndpoints))
				}

				if c.InputEndpoints[0].Name != "RDP" {
					return fmt.Errorf("Bad endpoint name: %s", c.InputEndpoints[0].Name)
				}

				if c.InputEndpoints[0].Port != 3389 {
					return fmt.Errorf("Bad endpoint port: %d", c.InputEndpoints[0].Port)
				}

				if len(c.SubnetNames) != 1 {
					return fmt.Errorf(
						"Instance %s has an unexpected number of associated subnets %d",
						dpmt.Name, len(c.SubnetNames))
				}

				if c.SubnetNames[0] != "subnet1" {
					return fmt.Errorf("Bad subnet: %s", c.SubnetNames[0])
				}

				if c.NetworkSecurityGroup != "terraform-security-group1" {
					return fmt.Errorf("Bad security group: %s", c.NetworkSecurityGroup)
				}
			}
		}

		return nil
	}
}

func testAccCheckAzureInstanceUpdatedAttributes(
	dpmt *virtualmachine.DeploymentResponse) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if dpmt.Name != "terraform-test1" {
			return fmt.Errorf("Bad name: %s", dpmt.Name)
		}

		if dpmt.VirtualNetworkName != "terraform-vnet-update-test" {
			return fmt.Errorf("Bad virtual network: %s", dpmt.VirtualNetworkName)
		}

		if len(dpmt.RoleList) != 1 {
			return fmt.Errorf(
				"Instance %s has an unexpected number of roles: %d", dpmt.Name, len(dpmt.RoleList))
		}

		if dpmt.RoleList[0].RoleSize != "Basic_A2" {
			return fmt.Errorf("Bad size: %s", dpmt.RoleList[0].RoleSize)
		}

		for _, c := range dpmt.RoleList[0].ConfigurationSets {
			if c.ConfigurationSetType == virtualmachine.ConfigurationSetTypeNetwork {
				if len(c.InputEndpoints) != 2 {
					return fmt.Errorf(
						"Instance %s has an unexpected number of endpoints %d",
						dpmt.Name, len(c.InputEndpoints))
				}

				if c.InputEndpoints[1].Name != "WINRM" {
					return fmt.Errorf("Bad endpoint name: %s", c.InputEndpoints[1].Name)
				}

				if c.InputEndpoints[1].Port != 5985 {
					return fmt.Errorf("Bad endpoint port: %d", c.InputEndpoints[1].Port)
				}

				if len(c.SubnetNames) != 1 {
					return fmt.Errorf(
						"Instance %s has an unexpected number of associated subnets %d",
						dpmt.Name, len(c.SubnetNames))
				}

				if c.SubnetNames[0] != "subnet1" {
					return fmt.Errorf("Bad subnet: %s", c.SubnetNames[0])
				}

				if c.NetworkSecurityGroup != "terraform-security-update-group2" {
					return fmt.Errorf("Bad security group: %s", c.NetworkSecurityGroup)
				}
			}
		}

		return nil
	}
}

func testAccCheckAzureInstanceDestroyed(hostedServiceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		hostedServiceClient := testAccProvider.Meta().(*Client).hostedServiceClient

		for _, rs := range s.RootModule().Resources {
			if rs.Type != "azure_instance" {
				continue
			}

			if rs.Primary.ID == "" {
				return fmt.Errorf("No instance ID is set")
			}

			// if not hosted service was provided; it means that we expect it
			// to be identical with the name of the instance; which is in the ID.
			var serviceName string
			if hostedServiceName == "" {
				serviceName = rs.Primary.ID
			} else {
				serviceName = hostedServiceName
			}

			_, err := hostedServiceClient.GetHostedService(serviceName)
			if err == nil {
				return fmt.Errorf("Instance %s still exists", rs.Primary.ID)
			}

			if !management.IsResourceNotFoundError(err) {
				return err
			}
		}

		return nil
	}
}

var testAccAzureInstance_basic = fmt.Sprintf(`
resource "azure_instance" "foo" {
    name = "%s"
    image = "Ubuntu Server 14.04 LTS"
    size = "Basic_A1"
    storage_service_name = "%s"
    location = "West US"
    username = "terraform"
    password = "Pass!admin123"

    endpoint {
        name = "SSH"
        protocol = "tcp"
        public_port = 22
        private_port = 22
    }
}`, instanceName, testAccStorageServiceName)

var testAccAzureInstance_separateHostedService = `
resource "azure_hosted_service" "foo" {
	name = "%s"
	location = "West US"
	ephemeral_contents = true
}

resource "azure_instance" "foo" {
    name = "%s"
    hosted_service_name = "${azure_hosted_service.foo.name}"
    image = "Ubuntu Server 14.04 LTS"
    size = "Basic_A1"
    storage_service_name = "%s"
    location = "West US"
    username = "terraform"
    password = "Pass!admin123"

    endpoint {
        name = "SSH"
        protocol = "tcp"
        public_port = 22
        private_port = 22
    }
}`

var testAccAzureInstance_advanced = fmt.Sprintf(`
resource "azure_virtual_network" "foo" {
    name = "terraform-vnet-advanced-test"
    address_space = ["10.1.2.0/24"]
		location = "West US"

		subnet {
        name = "subnet1"
				address_prefix = "10.1.2.0/25"
		}

		subnet {
        name = "subnet2"
				address_prefix = "10.1.2.128/25"
    }
}

resource "azure_security_group" "foo" {
    name = "terraform-security-group1"
    location = "West US"
}

resource "azure_security_group_rule" "foo" {
    name = "rdp"
    security_group_names = ["${azure_security_group.foo.name}"]
    priority = 101
    source_address_prefix = "*"
    source_port_range = "*"
    destination_address_prefix = "*"
    destination_port_range = "3389"
	action = "Deny"
	type = "Inbound"
    protocol = "TCP"
}

resource "azure_instance" "foo" {
    name = "terraform-test1"
    image = "Windows Server 2012 R2 Datacenter, January 2016"
    size = "Basic_A1"
    storage_service_name = "%s"
    location = "West US"
    time_zone = "America/Los_Angeles"
    subnet = "subnet1"
    virtual_network = "${azure_virtual_network.foo.name}"
    security_group = "${azure_security_group.foo.name}"
    username = "terraform"
    password = "Pass!admin123"

    endpoint {
        name = "RDP"
        protocol = "tcp"
        public_port = 3389
        private_port = 3389
    }
}`, testAccStorageServiceName)

var testAccAzureInstance_update = fmt.Sprintf(`
resource "azure_virtual_network" "foo" {
    name = "terraform-vnet-update-test"
    address_space = ["10.1.2.0/24"]
		location = "West US"

    subnet {
        name = "subnet1"
		address_prefix = "10.1.2.0/25"
	}

    subnet {
        name = "subnet2"
		address_prefix = "10.1.2.128/25"
    }
}

resource "azure_security_group" "foo" {
    name = "terraform-security-group1"
    location = "West US"
}

resource "azure_security_group_rule" "foo" {
    name = "rdp"
    security_group_names = ["${azure_security_group.foo.name}"]
    priority = 101
    source_address_prefix = "*"
    source_port_range = "*"
    destination_address_prefix = "*"
    destination_port_range = "3389"
	type = "Inbound"
	action = "Deny"
    protocol = "TCP"
}

resource "azure_security_group" "bar" {
    name = "terraform-security-update-group2"
    location = "West US"
}

resource "azure_security_group_rule" "bar" {
    name = "rdp"
    security_group_names = ["${azure_security_group.bar.name}"]
    priority = 101
    source_address_prefix = "192.168.0.0/24"
    source_port_range = "*"
    destination_address_prefix = "*"
    destination_port_range = "3389"
	type = "Inbound"
	action = "Deny"
    protocol = "TCP"
}

resource "azure_instance" "foo" {
    name = "terraform-test1"
    image = "Windows Server 2012 R2 Datacenter, January 2016"
    size = "Basic_A2"
    storage_service_name = "%s"
    location = "West US"
    time_zone = "America/Los_Angeles"
    subnet = "subnet1"
    virtual_network = "${azure_virtual_network.foo.name}"
    security_group = "${azure_security_group.bar.name}"
    username = "terraform"
    password = "Pass!admin123"

    endpoint {
        name = "RDP"
        protocol = "tcp"
        public_port = 3389
        private_port = 3389
    }

    endpoint {
        name = "WINRM"
        protocol = "tcp"
        public_port = 5985
        private_port = 5985
    }
}`, testAccStorageServiceName)
