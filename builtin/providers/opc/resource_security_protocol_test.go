package opc

import (
	"fmt"
	"testing"

	"github.com/hashicorp/go-oracle-terraform/compute"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccOPCSecurityProtocol_Basic(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccOPCSecurityProtocolBasic, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckSecurityProtocolDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSecurityProtocolExists,
				),
			},
		},
	})
}

func TestAccOPCSecurityProtocol_Complete(t *testing.T) {
	protocolResourceName := "opc_compute_security_protocol.test"
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccOPCSecurityProtocolComplete, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckSecurityProtocolDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSecurityProtocolExists,
					resource.TestCheckResourceAttr(protocolResourceName, "description", "Terraform Acceptance Test"),
					resource.TestCheckResourceAttr(protocolResourceName, "dst_ports.0", "2025-2030"),
					resource.TestCheckResourceAttr(protocolResourceName, "src_ports.0", "3025-3030"),
					resource.TestCheckResourceAttr(protocolResourceName, "ip_protocol", "tcp"),
				),
			},
		},
	})
}

func TestAccOPCSecurityProtocol_Update(t *testing.T) {
	protocolResourceName := "opc_compute_security_protocol.test"
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccOPCSecurityProtocolComplete, ri)
	config2 := fmt.Sprintf(testAccOPCSecurityProtocolUpdated, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckSecurityProtocolDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSecurityProtocolExists,
					resource.TestCheckResourceAttr(protocolResourceName, "description", "Terraform Acceptance Test"),
					resource.TestCheckResourceAttr(protocolResourceName, "dst_ports.0", "2025-2030"),
					resource.TestCheckResourceAttr(protocolResourceName, "src_ports.0", "3025-3030"),
					resource.TestCheckResourceAttr(protocolResourceName, "ip_protocol", "tcp"),
				),
			},
			{
				Config: config2,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSecurityProtocolExists,
					resource.TestCheckResourceAttr(protocolResourceName, "description", ""),
					resource.TestCheckResourceAttr(protocolResourceName, "dst_ports.1", "2040-2050"),
					resource.TestCheckResourceAttr(protocolResourceName, "src_ports.1", "3040-3050"),
					resource.TestCheckResourceAttr(protocolResourceName, "ip_protocol", "udp"),
				),
			},
		},
	})
}

func testAccCheckSecurityProtocolExists(s *terraform.State) error {
	client := testAccProvider.Meta().(*compute.Client).SecurityProtocols()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opc_compute_security_protocol" {
			continue
		}

		input := compute.GetSecurityProtocolInput{
			Name: rs.Primary.Attributes["name"],
		}
		if _, err := client.GetSecurityProtocol(&input); err != nil {
			return fmt.Errorf("Error retrieving state of Security Protocol %s: %s", input.Name, err)
		}
	}

	return nil
}

func testAccCheckSecurityProtocolDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*compute.Client).SecurityProtocols()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opc_compute_security_protocol" {
			continue
		}

		input := compute.GetSecurityProtocolInput{
			Name: rs.Primary.Attributes["name"],
		}
		if info, err := client.GetSecurityProtocol(&input); err == nil {
			return fmt.Errorf("Security Protocol %s still exists: %#v", input.Name, info)
		}
	}

	return nil
}

const testAccOPCSecurityProtocolBasic = `
resource "opc_compute_security_protocol" "test" {
	name        = "acc-security-protocol-%d"
  description = "Terraform Acceptance Test"
}
`

const testAccOPCSecurityProtocolComplete = `
resource "opc_compute_security_protocol" "test" {
  name        = "acc-security-protocol-%d"
  description = "Terraform Acceptance Test"
  dst_ports = ["2025-2030"]
  src_ports = ["3025-3030"]
  ip_protocol = "tcp"
}
`

const testAccOPCSecurityProtocolUpdated = `
resource "opc_compute_security_protocol" "test" {
  name        = "acc-security-protocol-%d"
  dst_ports   = ["2025-2030",	"2040-2050"]
  src_ports   = ["3025-3030",	"3040-3050"]
  ip_protocol = "udp"
}
`
