package opc

import (
	"fmt"
	"testing"

	"github.com/hashicorp/go-oracle-terraform/compute"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccOPCSecurityList_basic(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccOPCSecurityListBasic, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckSecurityListDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check:  testAccCheckSecurityListExists,
			},
		},
	})
}

func TestAccOPCSecurityList_complete(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccOPCSecurityListComplete, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckSecurityListDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check:  testAccCheckSecurityListExists,
			},
		},
	})
}

func testAccCheckSecurityListExists(s *terraform.State) error {
	client := testAccProvider.Meta().(*compute.Client).SecurityLists()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opc_compute_security_list" {
			continue
		}

		input := compute.GetSecurityListInput{
			Name: rs.Primary.Attributes["name"],
		}
		if _, err := client.GetSecurityList(&input); err != nil {
			return fmt.Errorf("Error retrieving state of Security List %s: %s", input.Name, err)
		}
	}

	return nil
}

func testAccCheckSecurityListDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*compute.Client).SecurityLists()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opc_compute_security_list" {
			continue
		}

		input := compute.GetSecurityListInput{
			Name: rs.Primary.Attributes["name"],
		}
		if info, err := client.GetSecurityList(&input); err == nil {
			return fmt.Errorf("Security List %s still exists: %#v", input.Name, info)
		}
	}

	return nil
}

const testAccOPCSecurityListBasic = `
resource "opc_compute_security_list" "test" {
  name                 = "acc-test-sec-list-%d"
  policy               = "PERMIT"
  outbound_cidr_policy = "DENY"
}
`

const testAccOPCSecurityListComplete = `
resource "opc_compute_security_list" "test" {
  name                 = "acc-test-sec-list-%d"
  description          = "Acceptance Test Security List Complete"
  policy               = "PERMIT"
  outbound_cidr_policy = "DENY"
}
`
