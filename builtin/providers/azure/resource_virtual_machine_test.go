package azure

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/MSOpenTech/azure-sdk-for-go/clients/vmClient"
)

func TestAccAzureVirtualMachine_Basic(t *testing.T) {
	var VMDeployment vmClient.VMDeployment

	// The VM name can only be used once globally within azure,
	// so we need to generate a random one
	rand.Seed(time.Now().UnixNano())
	vmName := fmt.Sprintf("tf-test-vm-%d", rand.Int31())

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAzureVirtualMachineDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckAzureVirtualMachineConfig_basic(vmName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAzureVirtualMachineExists("azure_virtual_machine.foobar", &VMDeployment),
					testAccCheckAzureVirtualMachineAttributes(&VMDeployment, vmName),
					resource.TestCheckResourceAttr(
						"azure_virtual_machine.foobar", "name", vmName),
					resource.TestCheckResourceAttr(
						"azure_virtual_machine.foobar", "location", "West US"),
					resource.TestCheckResourceAttr(
						"azure_virtual_machine.foobar", "image", "b39f27a8b8c64d52b05eac6a62ebad85__Ubuntu-14_04-LTS-amd64-server-20140724-en-us-30GB"),
					resource.TestCheckResourceAttr(
						"azure_virtual_machine.foobar", "size", "Basic_A1"),
					resource.TestCheckResourceAttr(
						"azure_virtual_machine.foobar", "username", "foobar"),
				),
			},
		},
	})
}

func TestAccAzureVirtualMachine_Endpoints(t *testing.T) {
	var VMDeployment vmClient.VMDeployment

	// The VM name can only be used once globally within azure,
	// so we need to generate a random one
	rand.Seed(time.Now().UnixNano())
	vmName := fmt.Sprintf("tf-test-vm-%d", rand.Int31())

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAzureVirtualMachineDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckAzureVirtualMachineConfig_endpoints(vmName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAzureVirtualMachineExists("azure_virtual_machine.foobar", &VMDeployment),
					testAccCheckAzureVirtualMachineAttributes(&VMDeployment, vmName),
					testAccCheckAzureVirtualMachineEndpoint(&VMDeployment, "tcp", 80),
				),
			},
		},
	})
}

func testAccCheckAzureVirtualMachineDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azure_virtual_machine" {
			continue
		}

		_, err := vmClient.GetVMDeployment(rs.Primary.ID, rs.Primary.ID)
		if err == nil {
			return fmt.Errorf("Azure Virtual Machine (%s) still exists", rs.Primary.ID)
		}
	}

	return nil
}

func testAccCheckAzureVirtualMachineExists(n string, VMDeployment *vmClient.VMDeployment) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Azure Virtual Machine ID is set")
		}

		retrieveVMDeployment, err := vmClient.GetVMDeployment(rs.Primary.ID, rs.Primary.ID)
		if err != nil {
			return err
		}

		if retrieveVMDeployment.Name != rs.Primary.ID {
			return fmt.Errorf("Azure Virtual Machine not found %s %s", VMDeployment.Name, rs.Primary.ID)
		}

		*VMDeployment = *retrieveVMDeployment

		return nil
	}
}

func testAccCheckAzureVirtualMachineAttributes(VMDeployment *vmClient.VMDeployment, vmName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if VMDeployment.Name != vmName {
			return fmt.Errorf("Bad name: %s != %s", VMDeployment.Name, vmName)
		}

		return nil
	}
}

func testAccCheckAzureVirtualMachineEndpoint(VMDeployment *vmClient.VMDeployment, protocol string, publicPort int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		roleInstances := VMDeployment.RoleInstanceList.RoleInstance
		if len(roleInstances) == 0 {
			return fmt.Errorf("Azure virtual machine does not have role instances")
		}

		for i := 0; i < len(roleInstances); i++ {
			instanceEndpoints := roleInstances[i].InstanceEndpoints.InstanceEndpoint
			if len(instanceEndpoints) == 0 {
				return fmt.Errorf("Azure virtual machine does not have endpoints")
			}
			endpointFound := 0
			for j := 0; i < len(instanceEndpoints); i++ {
				if instanceEndpoints[j].Protocol == protocol && instanceEndpoints[j].PublicPort == publicPort {
					endpointFound = 1
					break
				}
			}
			if endpointFound == 0 {
				return fmt.Errorf("Azure virtual machine does not have endpoint %s/%d", protocol, publicPort)
			}
		}

		return nil
	}
}

func testAccCheckAzureVirtualMachineConfig_basic(vmName string) string {
	return fmt.Sprintf(`
resource "azure_virtual_machine" "foobar" {
    name = "%s"
    location = "West US"
    image = "b39f27a8b8c64d52b05eac6a62ebad85__Ubuntu-14_04-LTS-amd64-server-20140724-en-us-30GB"
    size = "Basic_A1"
    username  = "foobar"
}
`, vmName)
}

func testAccCheckAzureVirtualMachineConfig_endpoints(vmName string) string {
	return fmt.Sprintf(`
resource "azure_virtual_machine" "foobar" {
    name = "%s"
    location = "West US"
    image = "b39f27a8b8c64d52b05eac6a62ebad85__Ubuntu-14_04-LTS-amd64-server-20140724-en-us-30GB"
    size = "Basic_A1"
    username  = "foobar"
    endpoint {
        name = "http"
        protocol = "tcp"
        port = 80
        local_port = 80
    }
}
`, vmName)
}
