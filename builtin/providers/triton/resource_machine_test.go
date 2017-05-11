package triton

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/joyent/triton-go"
)

func TestAccTritonMachine_basic(t *testing.T) {
	machineName := fmt.Sprintf("acctest-%d", acctest.RandInt())
	config := fmt.Sprintf(testAccTritonMachine_basic, machineName)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckTritonMachineDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckTritonMachineExists("triton_machine.test"),
					func(*terraform.State) error {
						time.Sleep(10 * time.Second)
						return nil
					},
				),
			},
		},
	})
}

func TestAccTritonMachine_dns(t *testing.T) {
	machineName := fmt.Sprintf("acctest-%d", acctest.RandInt())
	dns_output := fmt.Sprintf(testAccTritonMachine_dns, machineName)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckTritonMachineDestroy,
		Steps: []resource.TestStep{
			{
				Config: dns_output,
				Check: resource.ComposeTestCheckFunc(
					testCheckTritonMachineExists("triton_machine.test"),
					func(state *terraform.State) error {
						time.Sleep(10 * time.Second)
						log.Printf("[DEBUG] %s", spew.Sdump(state))
						return nil
					},
					resource.TestMatchOutput("domain_names", regexp.MustCompile(".*acctest-.*")),
				),
			},
		},
	})
}

func TestAccTritonMachine_nic(t *testing.T) {
	machineName := fmt.Sprintf("acctest-%d", acctest.RandInt())
	config := testAccTritonMachine_singleNIC(machineName, acctest.RandIntRange(1024, 2048), acctest.RandIntRange(0, 256))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckTritonMachineDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckTritonMachineExists("triton_machine.test"),
					func(*terraform.State) error {
						time.Sleep(10 * time.Second)
						return nil
					},
					testCheckTritonMachineHasFabric("triton_machine.test", "triton_fabric.test"),
				),
			},
		},
	})
}

func TestAccTritonMachine_addNIC(t *testing.T) {
	machineName := fmt.Sprintf("acctest-%d", acctest.RandInt())
	vlanNumber := acctest.RandIntRange(1024, 2048)
	subnetNumber := acctest.RandIntRange(0, 256)

	singleNICConfig := testAccTritonMachine_singleNIC(machineName, vlanNumber, subnetNumber)
	dualNICConfig := testAccTritonMachine_dualNIC(machineName, vlanNumber, subnetNumber)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckTritonMachineDestroy,
		Steps: []resource.TestStep{
			{
				Config: singleNICConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckTritonMachineExists("triton_machine.test"),
					func(*terraform.State) error {
						time.Sleep(10 * time.Second)
						return nil
					},
				),
			},
			{
				Config: dualNICConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckTritonMachineExists("triton_machine.test"),
					testCheckTritonMachineHasFabric("triton_machine.test", "triton_fabric.test_add"),
				),
			},
		},
	})
}

func testCheckTritonMachineExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}
		conn := testAccProvider.Meta().(*triton.Client)

		machine, err := conn.Machines().GetMachine(context.Background(), &triton.GetMachineInput{
			ID: rs.Primary.ID,
		})
		if err != nil {
			return fmt.Errorf("Bad: Check Machine Exists: %s", err)
		}

		if machine == nil {
			return fmt.Errorf("Bad: Machine %q does not exist", rs.Primary.ID)
		}

		return nil
	}
}

func testCheckTritonMachineHasFabric(name, fabricName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		machine, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		network, ok := s.RootModule().Resources[fabricName]
		if !ok {
			return fmt.Errorf("Not found: %s", fabricName)
		}
		conn := testAccProvider.Meta().(*triton.Client)

		nics, err := conn.Machines().ListNICs(context.Background(), &triton.ListNICsInput{
			MachineID: machine.Primary.ID,
		})
		if err != nil {
			return fmt.Errorf("Bad: Check NICs Exist: %s", err)
		}

		for _, nic := range nics {
			if nic.Network == network.Primary.ID {
				return nil
			}
		}

		return fmt.Errorf("Bad: Machine %q does not have Fabric %q", machine.Primary.ID, network.Primary.ID)
	}
}

func testCheckTritonMachineDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*triton.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "triton_machine" {
			continue
		}

		resp, err := conn.Machines().GetMachine(context.Background(), &triton.GetMachineInput{
			ID: rs.Primary.ID,
		})
		if err != nil {
			if triton.IsResourceNotFound(err) {
				return nil
			}
			return err
		}

		if resp != nil && resp.State != machineStateDeleted {
			return fmt.Errorf("Bad: Machine %q still exists", rs.Primary.ID)
		}
	}

	return nil
}

func TestAccTritonMachine_firewall(t *testing.T) {
	machineName := fmt.Sprintf("acctest-%d", acctest.RandInt())
	disabled_config := fmt.Sprintf(testAccTritonMachine_firewall_0, machineName)
	enabled_config := fmt.Sprintf(testAccTritonMachine_firewall_1, machineName)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckTritonMachineDestroy,
		Steps: []resource.TestStep{
			{
				Config: enabled_config,
				Check: resource.ComposeTestCheckFunc(
					testCheckTritonMachineExists("triton_machine.test"),
					resource.TestCheckResourceAttr(
						"triton_machine.test", "firewall_enabled", "true"),
				),
			},
			{
				Config: disabled_config,
				Check: resource.ComposeTestCheckFunc(
					testCheckTritonMachineExists("triton_machine.test"),
					resource.TestCheckResourceAttr(
						"triton_machine.test", "firewall_enabled", "false"),
				),
			},
			{
				Config: enabled_config,
				Check: resource.ComposeTestCheckFunc(
					testCheckTritonMachineExists("triton_machine.test"),
					resource.TestCheckResourceAttr(
						"triton_machine.test", "firewall_enabled", "true"),
				),
			},
		},
	})
}

func TestAccTritonMachine_metadata(t *testing.T) {
	machineName := fmt.Sprintf("acctest-%d", acctest.RandInt())
	basic := fmt.Sprintf(testAccTritonMachine_metadata_1, machineName)
	add_metadata := fmt.Sprintf(testAccTritonMachine_metadata_1, machineName)
	add_metadata_2 := fmt.Sprintf(testAccTritonMachine_metadata_2, machineName)
	add_metadata_3 := fmt.Sprintf(testAccTritonMachine_metadata_3, machineName)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckTritonMachineDestroy,
		Steps: []resource.TestStep{
			{
				Config: basic,
				Check: resource.ComposeTestCheckFunc(
					testCheckTritonMachineExists("triton_machine.test"),
				),
			},
			{
				Config: add_metadata,
				Check: resource.ComposeTestCheckFunc(
					testCheckTritonMachineExists("triton_machine.test"),
					resource.TestCheckResourceAttr(
						"triton_machine.test", "user_data", "hello"),
				),
			},
			{
				Config: add_metadata_2,
				Check: resource.ComposeTestCheckFunc(
					testCheckTritonMachineExists("triton_machine.test"),
					resource.TestCheckResourceAttr(
						"triton_machine.test",
						"tags.triton.cns.services", "test-cns-service"),
				),
			},
			{
				Config: add_metadata_3,
				Check: resource.ComposeTestCheckFunc(
					testCheckTritonMachineExists("triton_machine.test"),
					resource.TestCheckResourceAttr(
						"triton_machine.test",
						"tags.triton.cns.services", "test-cns-service"),
				),
			},
		},
	})
}

var testAccTritonMachine_basic = `
resource "triton_machine" "test" {
  name = "%s"
  package = "g4-general-4G"
  image = "fb5fe970-e6e4-11e6-9820-4b51be190db9"

  tags = {
	test = "hello!"
  }
}
`

var testAccTritonMachine_firewall_0 = `
resource "triton_machine" "test" {
  name = "%s"
  package = "g4-general-4G"
  image = "c20b4b7c-e1a6-11e5-9a4d-ef590901732e"

	firewall_enabled = 0
}
`
var testAccTritonMachine_firewall_1 = `
resource "triton_machine" "test" {
  name = "%s"
  package = "g4-general-4G"
  image = "fb5fe970-e6e4-11e6-9820-4b51be190db9"

	firewall_enabled = 1
}
`

var testAccTritonMachine_metadata_1 = `
resource "triton_machine" "test" {
  name = "%s"
  package = "g4-general-4G"
  image = "c20b4b7c-e1a6-11e5-9a4d-ef590901732e"

  user_data = "hello"

  tags = {
    test = "hello!"
	}
}
`
var testAccTritonMachine_metadata_2 = `
variable "tags" {
  default = {
    test = "hello!"
    triton.cns.services = "test-cns-service"
  }
}
resource "triton_machine" "test" {
  name = "%s"
  package = "g4-highcpu-128M"
  image = "fb5fe970-e6e4-11e6-9820-4b51be190db9"

  user_data = "hello"

  tags = "${var.tags}"
}
`
var testAccTritonMachine_metadata_3 = `
resource "triton_machine" "test" {
  name = "%s"
  package = "g4-highcpu-128M"
  image = "fb5fe970-e6e4-11e6-9820-4b51be190db9"

  user_data = "hello"

  tags = {
    test = "hello!"
    triton.cns.services = "test-cns-service"
  }
}
`
var testAccTritonMachine_singleNIC = func(name string, vlanNumber int, subnetNumber int) string {
	return fmt.Sprintf(`resource "triton_vlan" "test" {
	  vlan_id = %d
	  name = "%s-vlan"
	  description = "test vlan"
}

resource "triton_fabric" "test" {
	name = "%s-network"
	description = "test network"
	vlan_id = "${triton_vlan.test.vlan_id}"

	subnet = "10.%d.0.0/24"
	gateway = "10.%d.0.1"
	provision_start_ip = "10.%d.0.10"
	provision_end_ip = "10.%d.0.250"

	resolvers = ["8.8.8.8", "8.8.4.4"]
}

resource "triton_machine" "test" {
	name = "%s-instance"
	package = "g4-highcpu-128M"
	image = "fb5fe970-e6e4-11e6-9820-4b51be190db9"

	tags = {
		test = "Test"
	}

	nic {
		network = "${triton_fabric.test.id}"
	}
}`, vlanNumber, name, name, subnetNumber, subnetNumber, subnetNumber, subnetNumber, name)
}

var testAccTritonMachine_dualNIC = func(name string, vlanNumber, subnetNumber int) string {
	return fmt.Sprintf(`resource "triton_vlan" "test" {
	  vlan_id = %d
	  name = "%s-vlan"
	  description = "test vlan"
}

resource "triton_fabric" "test" {
	name = "%s-network"
	description = "test network"
	vlan_id = "${triton_vlan.test.vlan_id}"

	subnet = "10.%d.0.0/24"
	gateway = "10.%d.0.1"
	provision_start_ip = "10.%d.0.10"
	provision_end_ip = "10.%d.0.250"

	resolvers = ["8.8.8.8", "8.8.4.4"]
}

resource "triton_fabric" "test_add" {
	name = "%s-network-2"
	description = "test network 2"
	vlan_id = "${triton_vlan.test.vlan_id}"

	subnet = "172.23.%d.0/24"
	gateway = "172.23.%d.1"
	provision_start_ip = "172.23.%d.10"
	provision_end_ip = "172.23.%d.250"

	resolvers = ["8.8.8.8", "8.8.4.4"]
}

resource "triton_machine" "test" {
	name = "%s-instance"
	package = "g4-highcpu-128M"
	image = "fb5fe970-e6e4-11e6-9820-4b51be190db9"

	tags = {
		test = "Test"
	}

	nic {
		network = "${triton_fabric.test.id}"
	}
	nic {
		network = "${triton_fabric.test_add.id}"
	}
}`, vlanNumber, name, name, subnetNumber, subnetNumber, subnetNumber, subnetNumber, name, subnetNumber, subnetNumber, subnetNumber, subnetNumber, name)
}

var testAccTritonMachine_dns = `
provider "triton" {
}

resource "triton_machine" "test" {
  name = "%s"
  package = "g4-highcpu-128M"
  image = "fb5fe970-e6e4-11e6-9820-4b51be190db9"
}

output "domain_names" {
  value = "${join(", ", triton_machine.test.domain_names)}"
}
`
