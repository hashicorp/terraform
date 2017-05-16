package opc

import (
	"fmt"
	"testing"

	"github.com/hashicorp/go-oracle-terraform/compute"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccOPCACL_Basic(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccACLBasic, ri)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckACLDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckACLExists,
				),
			},
		},
	})
}

func TestAccOPCACL_Update(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccACLBasic, ri)
	updatedConfig := fmt.Sprintf(testAccACLDisabled, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckACLDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check:  testAccCheckACLExists,
			},
			{
				Config: updatedConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckACLExists,
					resource.TestCheckResourceAttr("opc_compute_acl.test", "enabled", "false"),
				),
			},
		},
	})
}

func testAccCheckACLExists(s *terraform.State) error {
	client := testAccProvider.Meta().(*compute.Client).ACLs()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opc_compute_acl" {
			continue
		}

		input := compute.GetACLInput{
			Name: rs.Primary.Attributes["name"],
		}
		if _, err := client.GetACL(&input); err != nil {
			return fmt.Errorf("Error retrieving state of ACL %s: %s", input.Name, err)
		}
	}

	return nil
}

func testAccCheckACLDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*compute.Client).ACLs()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opc_compute_acl" {
			continue
		}

		input := compute.GetACLInput{
			Name: rs.Primary.Attributes["name"],
		}
		if info, err := client.GetACL(&input); err == nil {
			return fmt.Errorf("ACL %s still exists: %#v", input.Name, info)
		}
	}

	return nil
}

var testAccACLBasic = `
resource "opc_compute_acl" "test" {
  name        = "test_acl-%d"
  description = "test acl"
}
`

var testAccACLDisabled = `
resource "opc_compute_acl" "test" {
  name        = "test_acl-%d"
  description = "test acl"
  enabled     = false
}
`
