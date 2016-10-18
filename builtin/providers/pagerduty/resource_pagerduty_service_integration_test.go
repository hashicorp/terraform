package pagerduty

import (
	"fmt"
	"testing"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccPagerDutyServiceIntegration_Basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPagerDutyServiceIntegrationDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckPagerDutyServiceIntegrationConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyServiceIntegrationExists("pagerduty_service_integration.foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_service_integration.foo", "name", "foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_service_integration.foo", "type", "generic_events_api_inbound_integration"),
				),
			},
			resource.TestStep{
				Config: testAccCheckPagerDutyServiceIntegrationConfigUpdated,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyServiceIntegrationExists("pagerduty_service_integration.foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_service_integration.foo", "name", "bar"),
					resource.TestCheckResourceAttr(
						"pagerduty_service_integration.foo", "type", "generic_events_api_inbound_integration"),
				),
			},
		},
	})
}

func testAccCheckPagerDutyServiceIntegrationDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*pagerduty.Client)
	for _, r := range s.RootModule().Resources {
		if r.Type != "pagerduty_service_integration" {
			continue
		}

		service, _ := s.RootModule().Resources["pagerduty_service.foo"]

		_, err := client.GetIntegration(service.Primary.ID, r.Primary.ID, pagerduty.GetIntegrationOptions{})

		if err == nil {
			return fmt.Errorf("Service Integration still exists")
		}

	}
	return nil
}

func testAccCheckPagerDutyServiceIntegrationExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No Service Integration ID is set")
		}

		service, _ := s.RootModule().Resources["pagerduty_service.foo"]

		client := testAccProvider.Meta().(*pagerduty.Client)

		found, err := client.GetIntegration(service.Primary.ID, rs.Primary.ID, pagerduty.GetIntegrationOptions{})
		if err != nil {
			return fmt.Errorf("Service integration not found: %v", rs.Primary.ID)
			// return err
		}

		if found.ID != rs.Primary.ID {
			return fmt.Errorf("Service Integration not found: %v - %v", rs.Primary.ID, found)
		}

		return nil
	}
}

const testAccCheckPagerDutyServiceIntegrationConfig = `
resource "pagerduty_user" "foo" {
  name        = "foo"
  email       = "foo@bar.com"
  color       = "green"
  role        = "user"
  job_title   = "foo"
  description = "foo"
}

resource "pagerduty_escalation_policy" "foo" {
  name        = "foo"
  description = "foo"
  num_loops   = 1

  escalation_rule {
    escalation_delay_in_minutes = 10

    target {
      type = "user_reference"
      id   = "${pagerduty_user.foo.id}"
    }
  }
}

resource "pagerduty_service" "foo" {
  name                    = "foo"
  description             = "foo"
  auto_resolve_timeout    = 1800
  acknowledgement_timeout = 1800
  escalation_policy       = "${pagerduty_escalation_policy.foo.id}"
}

resource "pagerduty_service_integration" "foo" {
  name    = "foo"
  type    = "generic_events_api_inbound_integration"
  service = "${pagerduty_service.foo.id}"
}
`

const testAccCheckPagerDutyServiceIntegrationConfigUpdated = `
resource "pagerduty_user" "foo" {
  name        = "foo"
  email       = "foo@bar.com"
  color       = "green"
  role        = "user"
  job_title   = "foo"
  description = "foo"
}

resource "pagerduty_escalation_policy" "foo" {
  name        = "bar"
  description = "bar"
  num_loops   = 2

  escalation_rule {
    escalation_delay_in_minutes = 10

    target {
      type = "user_reference"
      id   = "${pagerduty_user.foo.id}"
    }
  }
}

resource "pagerduty_service" "foo" {
  name                    = "bar"
  description             = "bar"
  auto_resolve_timeout    = 3600
  acknowledgement_timeout = 3600
  escalation_policy       = "${pagerduty_escalation_policy.foo.id}"
}

resource "pagerduty_service_integration" "foo" {
  name    = "bar"
  type    = "generic_events_api_inbound_integration"
  service = "${pagerduty_service.foo.id}"
}
`
