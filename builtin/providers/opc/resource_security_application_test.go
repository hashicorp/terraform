package opc

import (
	"fmt"
	"testing"

	"github.com/hashicorp/go-oracle-terraform/compute"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccOPCSecurityApplication_ICMP(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccOPCSecurityApplicationICMP, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccOPCCheckSecurityApplicationDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check:  testAccOPCCheckSecurityApplicationExists,
			},
		},
	})
}

func TestAccOPCSecurityApplication_TCP(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccOPCSecurityApplicationTCP, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccOPCCheckSecurityApplicationDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check:  testAccOPCCheckSecurityApplicationExists,
			},
		},
	})
}

func testAccOPCCheckSecurityApplicationExists(s *terraform.State) error {
	client := testAccProvider.Meta().(*compute.Client).SecurityApplications()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opc_compute_security_application" {
			continue
		}

		input := compute.GetSecurityApplicationInput{
			Name: rs.Primary.Attributes["name"],
		}
		if _, err := client.GetSecurityApplication(&input); err != nil {
			return fmt.Errorf("Error retrieving state of Security Application %s: %s", input.Name, err)
		}
	}

	return nil
}

func testAccOPCCheckSecurityApplicationDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*compute.Client).SecurityApplications()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opc_compute_security_application" {
			continue
		}

		input := compute.GetSecurityApplicationInput{
			Name: rs.Primary.Attributes["name"],
		}
		if info, err := client.GetSecurityApplication(&input); err == nil {
			return fmt.Errorf("Security Application %s still exists: %#v", input.Name, info)
		}
	}

	return nil
}

const testAccOPCSecurityApplicationTCP = `
resource "opc_compute_security_application" "test" {
	name        = "acc-security-application-tcp-%d"
	protocol    = "tcp"
	dport       = "8080"
	description = "Terraform Acceptance Test"
}
`

const testAccOPCSecurityApplicationICMP = `
resource "opc_compute_security_application" "test" {
	name        = "acc-security-application-tcp-%d"
	protocol    = "icmp"
	icmptype    = "echo"
	description = "Terraform Acceptance Test"
}
`
