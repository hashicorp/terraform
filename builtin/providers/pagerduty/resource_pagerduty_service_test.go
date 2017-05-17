package pagerduty

import (
	"fmt"
	"testing"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccPagerDutyService_Basic(t *testing.T) {
	username := fmt.Sprintf("tf-%s", acctest.RandString(5))
	email := fmt.Sprintf("%s@foo.com", username)
	escalationPolicy := fmt.Sprintf("tf-%s", acctest.RandString(5))
	service := fmt.Sprintf("tf-%s", acctest.RandString(5))
	serviceUpdated := fmt.Sprintf("tf-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPagerDutyServiceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckPagerDutyServiceConfig(username, email, escalationPolicy, service),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyServiceExists("pagerduty_service.foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "name", service),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "description", "foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "auto_resolve_timeout", "1800"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "acknowledgement_timeout", "1800"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "incident_urgency_rule.#", "1"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "incident_urgency_rule.0.urgency", "high"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "incident_urgency_rule.0.type", "constant"),
				),
			},
			{
				Config: testAccCheckPagerDutyServiceConfigUpdated(username, email, escalationPolicy, serviceUpdated),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyServiceExists("pagerduty_service.foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "name", serviceUpdated),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "description", "bar"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "auto_resolve_timeout", "3600"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "acknowledgement_timeout", "3600"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "incident_urgency_rule.#", "1"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "incident_urgency_rule.0.urgency", "high"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "incident_urgency_rule.0.type", "constant"),
				),
			},
		},
	})
}

func TestAccPagerDutyService_BasicWithIncidentUrgencyRules(t *testing.T) {
	username := fmt.Sprintf("tf-%s", acctest.RandString(5))
	email := fmt.Sprintf("%s@foo.com", username)
	escalationPolicy := fmt.Sprintf("tf-%s", acctest.RandString(5))
	service := fmt.Sprintf("tf-%s", acctest.RandString(5))
	serviceUpdated := fmt.Sprintf("tf-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPagerDutyServiceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckPagerDutyServiceWithIncidentUrgencyRulesConfig(username, email, escalationPolicy, service),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyServiceExists("pagerduty_service.foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "name", service),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "description", "foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "auto_resolve_timeout", "1800"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "acknowledgement_timeout", "1800"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "incident_urgency_rule.#", "1"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "incident_urgency_rule.0.during_support_hours.#", "1"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "incident_urgency_rule.0.during_support_hours.0.type", "constant"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "incident_urgency_rule.0.during_support_hours.0.urgency", "high"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "incident_urgency_rule.0.outside_support_hours.#", "1"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "incident_urgency_rule.0.outside_support_hours.0.type", "constant"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "incident_urgency_rule.0.outside_support_hours.0.urgency", "low"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "incident_urgency_rule.0.type", "use_support_hours"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "scheduled_actions.#", "1"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "scheduled_actions.0.at.#", "1"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "scheduled_actions.0.at.0.name", "support_hours_start"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "scheduled_actions.0.to_urgency", "high"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "scheduled_actions.0.type", "urgency_change"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "support_hours.#", "1"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "support_hours.0.days_of_week.#", "5"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "support_hours.0.days_of_week.0", "1"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "support_hours.0.days_of_week.1", "2"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "support_hours.0.days_of_week.2", "3"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "support_hours.0.days_of_week.3", "4"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "support_hours.0.days_of_week.4", "5"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "support_hours.0.end_time", "17:00:00"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "support_hours.0.start_time", "09:00:00"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "support_hours.0.time_zone", "America/Lima"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "support_hours.0.type", "fixed_time_per_day"),
				),
			},
			{
				Config: testAccCheckPagerDutyServiceWithIncidentUrgencyRulesConfigUpdated(username, email, escalationPolicy, serviceUpdated),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyServiceExists("pagerduty_service.foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "name", serviceUpdated),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "description", "bar bar bar"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "auto_resolve_timeout", "3600"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "acknowledgement_timeout", "3600"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "incident_urgency_rule.#", "1"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "incident_urgency_rule.0.during_support_hours.#", "1"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "incident_urgency_rule.0.during_support_hours.0.type", "constant"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "incident_urgency_rule.0.during_support_hours.0.urgency", "high"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "incident_urgency_rule.0.outside_support_hours.#", "1"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "incident_urgency_rule.0.outside_support_hours.0.type", "constant"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "incident_urgency_rule.0.outside_support_hours.0.urgency", "low"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "incident_urgency_rule.0.type", "use_support_hours"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "scheduled_actions.#", "1"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "scheduled_actions.0.at.#", "1"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "scheduled_actions.0.at.0.name", "support_hours_start"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "scheduled_actions.0.to_urgency", "high"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "scheduled_actions.0.type", "urgency_change"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "support_hours.#", "1"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "support_hours.0.days_of_week.#", "5"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "support_hours.0.days_of_week.0", "1"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "support_hours.0.days_of_week.1", "2"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "support_hours.0.days_of_week.2", "3"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "support_hours.0.days_of_week.3", "4"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "support_hours.0.days_of_week.4", "5"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "support_hours.0.end_time", "17:00:00"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "support_hours.0.start_time", "09:00:00"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "support_hours.0.time_zone", "America/Lima"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "support_hours.0.type", "fixed_time_per_day"),
				),
			},
		},
	})
}

func TestAccPagerDutyService_FromBasicToCustomIncidentUrgencyRules(t *testing.T) {
	username := fmt.Sprintf("tf-%s", acctest.RandString(5))
	email := fmt.Sprintf("%s@foo.com", username)
	escalationPolicy := fmt.Sprintf("tf-%s", acctest.RandString(5))
	service := fmt.Sprintf("tf-%s", acctest.RandString(5))
	serviceUpdated := fmt.Sprintf("tf-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPagerDutyServiceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckPagerDutyServiceConfig(username, email, escalationPolicy, service),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyServiceExists("pagerduty_service.foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "name", service),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "description", "foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "auto_resolve_timeout", "1800"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "acknowledgement_timeout", "1800"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "incident_urgency_rule.#", "1"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "incident_urgency_rule.0.urgency", "high"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "incident_urgency_rule.0.type", "constant"),
				),
			},
			{
				Config: testAccCheckPagerDutyServiceWithIncidentUrgencyRulesConfigUpdated(username, email, escalationPolicy, serviceUpdated),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyServiceExists("pagerduty_service.foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "name", serviceUpdated),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "description", "bar bar bar"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "auto_resolve_timeout", "3600"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "acknowledgement_timeout", "3600"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "incident_urgency_rule.#", "1"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "incident_urgency_rule.0.during_support_hours.#", "1"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "incident_urgency_rule.0.during_support_hours.0.type", "constant"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "incident_urgency_rule.0.during_support_hours.0.urgency", "high"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "incident_urgency_rule.0.outside_support_hours.#", "1"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "incident_urgency_rule.0.outside_support_hours.0.type", "constant"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "incident_urgency_rule.0.outside_support_hours.0.urgency", "low"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "incident_urgency_rule.0.type", "use_support_hours"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "scheduled_actions.#", "1"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "scheduled_actions.0.at.#", "1"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "scheduled_actions.0.at.0.name", "support_hours_start"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "scheduled_actions.0.to_urgency", "high"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "scheduled_actions.0.type", "urgency_change"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "support_hours.#", "1"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "support_hours.0.days_of_week.#", "5"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "support_hours.0.days_of_week.0", "1"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "support_hours.0.days_of_week.1", "2"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "support_hours.0.days_of_week.2", "3"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "support_hours.0.days_of_week.3", "4"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "support_hours.0.days_of_week.4", "5"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "support_hours.0.end_time", "17:00:00"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "support_hours.0.start_time", "09:00:00"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "support_hours.0.time_zone", "America/Lima"),
					resource.TestCheckResourceAttr(
						"pagerduty_service.foo", "support_hours.0.type", "fixed_time_per_day"),
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

		_, err := client.GetService(r.Primary.ID, &pagerduty.GetServiceOptions{})

		if err == nil {
			return fmt.Errorf("Service still exists")
		}

	}
	return nil
}

func testAccCheckPagerDutyServiceExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Service ID is set")
		}

		client := testAccProvider.Meta().(*pagerduty.Client)

		found, err := client.GetService(rs.Primary.ID, &pagerduty.GetServiceOptions{})
		if err != nil {
			return err
		}

		if found.ID != rs.Primary.ID {
			return fmt.Errorf("Service not found: %v - %v", rs.Primary.ID, found)
		}

		return nil
	}
}

func testAccCheckPagerDutyServiceConfig(username, email, escalationPolicy, service string) string {
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
	description             = "foo"
	auto_resolve_timeout    = 1800
	acknowledgement_timeout = 1800
	escalation_policy       = "${pagerduty_escalation_policy.foo.id}"
	incident_urgency_rule {
		type    = "constant"
		urgency = "high"
	}
}
`, username, email, escalationPolicy, service)
}

func testAccCheckPagerDutyServiceConfigUpdated(username, email, escalationPolicy, service string) string {
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
`, username, email, escalationPolicy, service)
}

func testAccCheckPagerDutyServiceWithIncidentUrgencyRulesConfig(username, email, escalationPolicy, service string) string {
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
	description             = "foo"
	auto_resolve_timeout    = 1800
	acknowledgement_timeout = 1800
	escalation_policy       = "${pagerduty_escalation_policy.foo.id}"

	incident_urgency_rule {
		type = "use_support_hours"

		during_support_hours {
			type    = "constant"
			urgency = "high"
		}
		outside_support_hours {
			type    = "constant"
			urgency = "low"
		}
	}

	support_hours = [{
		type         = "fixed_time_per_day"
		time_zone    = "America/Lima"
		start_time   = "09:00:00"
		end_time     = "17:00:00"
		days_of_week = [ 1, 2, 3, 4, 5 ]
	}]

	scheduled_actions {
		type = "urgency_change"
		to_urgency = "high"
		at {
			type = "named_time",
			name = "support_hours_start"
		}
	}
}
`, username, email, escalationPolicy, service)
}

func testAccCheckPagerDutyServiceWithIncidentUrgencyRulesConfigUpdated(username, email, escalationPolicy, service string) string {
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
	description             = "bar bar bar"
	auto_resolve_timeout    = 3600
	acknowledgement_timeout = 3600
	escalation_policy       = "${pagerduty_escalation_policy.foo.id}"

	incident_urgency_rule {
		type = "use_support_hours"
		during_support_hours {
			type    = "constant"
			urgency = "high"
		}
		outside_support_hours {
			type    = "constant"
			urgency = "low"
		}
	}

	support_hours = [{
		type         = "fixed_time_per_day"
		time_zone    = "America/Lima"
		start_time   = "09:00:00"
		end_time     = "17:00:00"
		days_of_week = [ 1, 2, 3, 4, 5 ]
	}]

	scheduled_actions {
		type = "urgency_change"
		to_urgency = "high"
		at {
			type = "named_time",
			name = "support_hours_start"
		}
	}
}
`, username, email, escalationPolicy, service)
}
