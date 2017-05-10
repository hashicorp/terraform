package pagerduty

import (
	"fmt"
	"testing"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccPagerDutyServiceIntegration_Basic(t *testing.T) {
	username := fmt.Sprintf("tf-%s", acctest.RandString(5))
	email := fmt.Sprintf("%s@foo.com", username)
	escalationPolicy := fmt.Sprintf("tf-%s", acctest.RandString(5))
	service := fmt.Sprintf("tf-%s", acctest.RandString(5))
	serviceIntegration := fmt.Sprintf("tf-%s", acctest.RandString(5))
	serviceIntegrationUpdated := fmt.Sprintf("tf-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPagerDutyServiceIntegrationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckPagerDutyServiceIntegrationConfig(username, email, escalationPolicy, service, serviceIntegration),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyServiceIntegrationExists("pagerduty_service_integration.foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_service_integration.foo", "name", serviceIntegration),
					resource.TestCheckResourceAttr(
						"pagerduty_service_integration.foo", "type", "generic_events_api_inbound_integration"),
					resource.TestCheckResourceAttr(
						"pagerduty_service_integration.foo", "vendor", "PAM4FGS"),
				),
			},
			{
				Config: testAccCheckPagerDutyServiceIntegrationConfigUpdated(username, email, escalationPolicy, service, serviceIntegrationUpdated),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyServiceIntegrationExists("pagerduty_service_integration.foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_service_integration.foo", "name", serviceIntegrationUpdated),
					resource.TestCheckResourceAttr(
						"pagerduty_service_integration.foo", "type", "generic_events_api_inbound_integration"),
					resource.TestCheckResourceAttr(
						"pagerduty_service_integration.foo", "vendor", "PAM4FGS"),
				),
			},
		},
	})
}

func TestAccPagerDutyServiceIntegrationGeneric_Basic(t *testing.T) {
	username := fmt.Sprintf("tf-%s", acctest.RandString(5))
	email := fmt.Sprintf("%s@foo.com", username)
	escalationPolicy := fmt.Sprintf("tf-%s", acctest.RandString(5))
	service := fmt.Sprintf("tf-%s", acctest.RandString(5))
	serviceIntegration := fmt.Sprintf("tf-%s", acctest.RandString(5))
	serviceIntegrationUpdated := fmt.Sprintf("tf-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPagerDutyServiceIntegrationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckPagerDutyServiceIntegrationGenericConfig(username, email, escalationPolicy, service, serviceIntegration),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyServiceIntegrationExists("pagerduty_service_integration.foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_service_integration.foo", "name", serviceIntegration),
					resource.TestCheckResourceAttr(
						"pagerduty_service_integration.foo", "type", "generic_events_api_inbound_integration"),
				),
			},
			{
				Config: testAccCheckPagerDutyServiceIntegrationGenericConfigUpdated(username, email, escalationPolicy, service, serviceIntegrationUpdated),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyServiceIntegrationExists("pagerduty_service_integration.foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_service_integration.foo", "name", serviceIntegrationUpdated),
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
		}

		if found.ID != rs.Primary.ID {
			return fmt.Errorf("Service Integration not found: %v - %v", rs.Primary.ID, found)
		}

		return nil
	}
}

func testAccCheckPagerDutyServiceIntegrationConfig(username, email, escalationPolicy, service, serviceIntegration string) string {
	return fmt.Sprintf(`
resource "pagerduty_user" "foo" {
  name        = "%s"
  email       = "%s"
}

resource "pagerduty_escalation_policy" "foo" {
  name        = "%s"
  description = "foo"
  num_loops   = 1

  rule {
    escalation_delay_in_minutes = 10

    target {
      type = "user_reference"
      id   = "${pagerduty_user.foo.id}"
    }
  }
}

resource "pagerduty_service" "foo" {
  name                    = "%s"
  description             = "foo"
  auto_resolve_timeout    = 1800
  acknowledgement_timeout = 1800
  escalation_policy       = "${pagerduty_escalation_policy.foo.id}"

  incident_urgency_rule {
    type = "constant"
    urgency = "high"
  }
}

data "pagerduty_vendor" "datadog" {
  name = "datadog"
}

resource "pagerduty_service_integration" "foo" {
  name    = "%s"
  service = "${pagerduty_service.foo.id}"
  vendor  = "${data.pagerduty_vendor.datadog.id}"
}
`, username, email, escalationPolicy, service, serviceIntegration)
}

func testAccCheckPagerDutyServiceIntegrationConfigUpdated(username, email, escalationPolicy, service, serviceIntegration string) string {
	return fmt.Sprintf(`
resource "pagerduty_user" "foo" {
  name        = "%s"
  email       = "%s"
  color       = "green"
  role        = "user"
  job_title   = "foo"
  description = "foo"
}

resource "pagerduty_escalation_policy" "foo" {
  name        = "%s"
  description = "bar"
  num_loops   = 2

  rule {
    escalation_delay_in_minutes = 10

    target {
      type = "user_reference"
      id   = "${pagerduty_user.foo.id}"
    }
  }
}

resource "pagerduty_service" "foo" {
  name                    = "%s"
  description             = "bar"
  auto_resolve_timeout    = 3600
  acknowledgement_timeout = 3600
  escalation_policy       = "${pagerduty_escalation_policy.foo.id}"

  incident_urgency_rule {
    type    = "constant"
    urgency = "high"
  }
}

data "pagerduty_vendor" "datadog" {
  name = "datadog"
}

resource "pagerduty_service_integration" "foo" {
  name    = "%s"
  service = "${pagerduty_service.foo.id}"
  vendor  = "${data.pagerduty_vendor.datadog.id}"
}
`, username, email, escalationPolicy, service, serviceIntegration)
}

func testAccCheckPagerDutyServiceIntegrationGenericConfig(username, email, escalationPolicy, service, serviceIntegration string) string {
	return fmt.Sprintf(`
resource "pagerduty_user" "foo" {
  name        = "%s"
  email       = "%s"
}

resource "pagerduty_escalation_policy" "foo" {
  name        = "%s"
  description = "foo"
  num_loops   = 1

  rule {
    escalation_delay_in_minutes = 10

    target {
      type = "user_reference"
      id   = "${pagerduty_user.foo.id}"
    }
  }
}

resource "pagerduty_service" "foo" {
  name                    = "%s"
  description             = "foo"
  auto_resolve_timeout    = 1800
  acknowledgement_timeout = 1800
  escalation_policy       = "${pagerduty_escalation_policy.foo.id}"

  incident_urgency_rule {
    type = "constant"
    urgency = "high"
  }
}

resource "pagerduty_service_integration" "foo" {
  name    = "%s"
  service = "${pagerduty_service.foo.id}"
  type    = "generic_events_api_inbound_integration"
}
`, username, email, escalationPolicy, service, serviceIntegration)
}

func testAccCheckPagerDutyServiceIntegrationGenericConfigUpdated(username, email, escalationPolicy, service, serviceIntegration string) string {
	return fmt.Sprintf(`
resource "pagerduty_user" "foo" {
  name        = "%s"
  email       = "%s"
  color       = "green"
  role        = "user"
  job_title   = "foo"
  description = "foo"
}

resource "pagerduty_escalation_policy" "foo" {
  name        = "%s"
  description = "bar"
  num_loops   = 2

  rule {
    escalation_delay_in_minutes = 10

    target {
      type = "user_reference"
      id   = "${pagerduty_user.foo.id}"
    }
  }
}

resource "pagerduty_service" "foo" {
  name                    = "%s"
  description             = "bar"
  auto_resolve_timeout    = 3600
  acknowledgement_timeout = 3600
  escalation_policy       = "${pagerduty_escalation_policy.foo.id}"

  incident_urgency_rule {
    type    = "constant"
    urgency = "high"
  }
}

resource "pagerduty_service_integration" "foo" {
  name    = "%s"
  service = "${pagerduty_service.foo.id}"
  type    = "generic_events_api_inbound_integration"
}
`, username, email, escalationPolicy, service, serviceIntegration)
}
