package pagerduty

import (
	"fmt"
	"testing"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccPagerDutyService_Basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPagerDutyServiceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckPagerDutyServiceConfig(escalationPolicyID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyServiceExists("pagerduty_service.foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "name", "foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "description", "foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "auto_resolve_timeout", "1800"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "acknowledgement_timeout", "1800"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "escalation_policy", escalationPolicyID),
				),
			},
			resource.TestStep{
				Config: testAccCheckPagerDutyServiceConfigUpdated(escalationPolicyID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyServiceExists("pagerduty_service.foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "name", "bar"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "description", "bar"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "auto_resolve_timeout", "3600"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "acknowledgement_timeout", "3600"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "escalation_policy", escalationPolicyID),
				),
			},
		},
	})
}

func testAccCheckPagerDutyServiceDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*pagerduty.Client)
	for _, r := range s.RootModule().Resources {
		if r.Type != "pagerduty_service" {
			continue
		}

		_, err := client.GetService(r.Primary.ID, pagerduty.GetServiceOptions{})

		if err == nil {
			return fmt.Errorf("Service still exists")
		}

	}
	return nil
}

func testAccCheckPagerDutyServiceExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*pagerduty.Client)
		for _, r := range s.RootModule().Resources {
			if _, err := client.GetService(r.Primary.ID, pagerduty.GetServiceOptions{}); err != nil {
				return fmt.Errorf("Received an error retrieving service %s ID: %s", err, r.Primary.ID)
			}
		}
		return nil
	}
}

func testAccCheckPagerDutyServiceConfig(id string) string {
	return fmt.Sprintf(`
		resource "pagerduty_service" "foo" {
		  name                    = "foo"
		  description             = "foo"
			auto_resolve_timeout    = 1800
			acknowledgement_timeout = 1800
			escalation_policy       = "%s"
		}
	`, escalationPolicyID)
}

func testAccCheckPagerDutyServiceConfigUpdated(id string) string {
	return fmt.Sprintf(`
resource "pagerduty_service" "foo" {
  name                    = "bar"
  description             = "bar"
	auto_resolve_timeout    = 3600
	acknowledgement_timeout = 3600
	escalation_policy       = "%s"
}
`, escalationPolicyID)
}
