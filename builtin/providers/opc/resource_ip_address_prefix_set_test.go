package opc

import (
	"fmt"
	"testing"

	"github.com/hashicorp/go-oracle-terraform/compute"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccOPCIPAddressPrefixSet_Basic(t *testing.T) {
	rInt := acctest.RandInt()
	resourceName := "opc_compute_ip_address_prefix_set.test"
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckIPAddressPrefixSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccIPAddressPrefixSetBasic(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIPAddressPrefixSetExists,
					resource.TestCheckResourceAttr(
						resourceName, "tags.#", "2"),
					resource.TestCheckResourceAttr(
						resourceName, "prefixes.#", "2"),
				),
			},
			{
				Config: testAccIPAddressPrefixSetBasic_Update(rInt),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						resourceName, "tags.#", "1"),
					resource.TestCheckResourceAttr(
						resourceName, "prefixes.0", "171.120.0.0/24"),
				),
			},
		},
	})
}

func testAccCheckIPAddressPrefixSetExists(s *terraform.State) error {
	client := testAccProvider.Meta().(*compute.Client).IPAddressPrefixSets()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opc_compute_ip_address_prefix_set" {
			continue
		}

		input := compute.GetIPAddressPrefixSetInput{
			Name: rs.Primary.Attributes["name"],
		}
		if _, err := client.GetIPAddressPrefixSet(&input); err != nil {
			return fmt.Errorf("Error retrieving state of IP Address Prefix Set %s: %s", input.Name, err)
		}
	}

	return nil
}

func testAccCheckIPAddressPrefixSetDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*compute.Client).IPAddressPrefixSets()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opc_compute_ip_address_prefix_set" {
			continue
		}

		input := compute.GetIPAddressPrefixSetInput{
			Name: rs.Primary.Attributes["name"],
		}
		if info, err := client.GetIPAddressPrefixSet(&input); err == nil {
			return fmt.Errorf("IP Address Prefix Set %s still exists: %#v", input.Name, info)
		}
	}

	return nil
}

func testAccIPAddressPrefixSetBasic(rInt int) string {
	return fmt.Sprintf(`
resource "opc_compute_ip_address_prefix_set" "test" {
  name = "testing-acc-%d"
	prefixes = ["172.120.0.0/24", "192.168.0.0/16"]
  description = "acctesting ip address prefix test %d"
  tags = ["tag1", "tag2"]
}`, rInt, rInt)
}

func testAccIPAddressPrefixSetBasic_Update(rInt int) string {
	return fmt.Sprintf(`
resource "opc_compute_ip_address_prefix_set" "test" {
  name = "testing-acc-%d"
  description = "acctesting ip address prefix test updated %d"
	prefixes = ["171.120.0.0/24", "192.168.0.0/16"]
  tags = ["tag1"]
}`, rInt, rInt)
}
