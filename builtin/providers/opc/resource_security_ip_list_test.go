package opc

import (
	"fmt"
	"testing"

	"github.com/hashicorp/go-oracle-terraform/compute"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccOPCSecurityIPList_Basic(t *testing.T) {
	listResourceName := "opc_compute_security_ip_list.test"
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccOPCSecurityIPListBasic, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckSecurityIPListDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSecurityIPListExists,
					resource.TestCheckResourceAttr(listResourceName, "ip_entries.0", "192.168.0.1"),
					resource.TestCheckResourceAttr(listResourceName, "ip_entries.1", "192.168.0.2"),
				),
			},
		},
	})
}

func TestAccOPCSecurityIPList_Updated(t *testing.T) {
	listResourceName := "opc_compute_security_ip_list.test"
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccOPCSecurityIPListBasic, ri)
	config2 := fmt.Sprintf(testAccOPCSecurityIPListUpdated, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckSecurityIPListDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSecurityIPListExists,
					resource.TestCheckResourceAttr(listResourceName, "description", "Terraform Acceptance Test"),
				),
			},
			{
				Config: config2,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSecurityIPListExists,
					resource.TestCheckResourceAttr(listResourceName, "description", ""),
					resource.TestCheckResourceAttr(listResourceName, "ip_entries.0", "192.168.0.1"),
					resource.TestCheckResourceAttr(listResourceName, "ip_entries.1", "192.168.0.3"),
				),
			},
		},
	})
}

func testAccCheckSecurityIPListExists(s *terraform.State) error {
	client := testAccProvider.Meta().(*compute.Client).SecurityIPLists()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opc_compute_security_ip_list" {
			continue
		}

		input := compute.GetSecurityIPListInput{
			Name: rs.Primary.Attributes["name"],
		}
		if _, err := client.GetSecurityIPList(&input); err != nil {
			return fmt.Errorf("Error retrieving state of Security IP List %s: %s", input.Name, err)
		}
	}

	return nil
}

func testAccCheckSecurityIPListDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*compute.Client).SecurityIPLists()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opc_compute_security_ip_list" {
			continue
		}

		input := compute.GetSecurityIPListInput{
			Name: rs.Primary.Attributes["name"],
		}
		if info, err := client.GetSecurityIPList(&input); err == nil {
			return fmt.Errorf("Security IP List %s still exists: %#v", input.Name, info)
		}
	}

	return nil
}

const testAccOPCSecurityIPListBasic = `
resource "opc_compute_security_ip_list" "test" {
	name        = "acc-security-application-tcp-%d"
  ip_entries = ["192.168.0.1", "192.168.0.2"]
	description = "Terraform Acceptance Test"
}
`

const testAccOPCSecurityIPListUpdated = `
resource "opc_compute_security_ip_list" "test" {
	name        = "acc-security-application-tcp-%d"
  ip_entries = ["192.168.0.1", "192.168.0.3"]
}
`
