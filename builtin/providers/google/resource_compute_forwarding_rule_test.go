package google

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccComputeForwardingRule_basic(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeForwardingRuleDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeForwardingRule_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeForwardingRuleExists(
						"google_compute_forwarding_rule.foobar"),
				),
			},
		},
	})
}

func TestAccComputeForwardingRule_ip(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeForwardingRuleDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeForwardingRule_ip,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeForwardingRuleExists(
						"google_compute_forwarding_rule.foobar"),
				),
			},
		},
	})
}

func testAccCheckComputeForwardingRuleDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "google_compute_forwarding_rule" {
			continue
		}

		_, err := config.clientCompute.ForwardingRules.Get(
			config.Project, config.Region, rs.Primary.ID).Do()
		if err == nil {
			return fmt.Errorf("ForwardingRule still exists")
		}
	}

	return nil
}

func testAccCheckComputeForwardingRuleExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)

		found, err := config.clientCompute.ForwardingRules.Get(
			config.Project, config.Region, rs.Primary.ID).Do()
		if err != nil {
			return err
		}

		if found.Name != rs.Primary.ID {
			return fmt.Errorf("ForwardingRule not found")
		}

		return nil
	}
}

const testAccComputeForwardingRule_basic = `
resource "google_compute_target_pool" "foobar-tp" {
	description = "Resource created for Terraform acceptance testing"
	instances = ["us-central1-a/foo", "us-central1-b/bar"]
	name = "terraform-test"
}
resource "google_compute_forwarding_rule" "foobar" {
	description = "Resource created for Terraform acceptance testing"
	ip_protocol = "UDP"
    name = "terraform-test"
    port_range = "80-81"
    target = "${google_compute_target_pool.foobar-tp.self_link}"
}
`

const testAccComputeForwardingRule_ip = `
resource "google_compute_address" "foo" {
    name = "foo"
}
resource "google_compute_target_pool" "foobar-tp" {
	description = "Resource created for Terraform acceptance testing"
	instances = ["us-central1-a/foo", "us-central1-b/bar"]
	name = "terraform-test"
}
resource "google_compute_forwarding_rule" "foobar" {
	description = "Resource created for Terraform acceptance testing"
	ip_address = "${google_compute_address.foo.address}"
	ip_protocol = "TCP"
    name = "terraform-test"
    port_range = "80-81"
    target = "${google_compute_target_pool.foobar-tp.self_link}"
}
`
