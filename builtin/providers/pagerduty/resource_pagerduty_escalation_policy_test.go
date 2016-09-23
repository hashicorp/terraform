package pagerduty

import (
	"fmt"
	"testing"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccPagerDutyEscalationPolicy_Basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPagerDutyEscalationPolicyDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckPagerDutyEscalationPolicyConfig(userID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyEscalationPolicyExists("pagerduty_escalation_policy.foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_escalation_policy.foo", "name", "foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_escalation_policy.foo", "description", "foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_escalation_policy.foo", "num_loops", "1"),
					resource.TestCheckResourceAttr(
						"pagerduty_escalation_policy.foo", "escalation_rule.#", "1"),
					resource.TestCheckResourceAttr(
						"pagerduty_escalation_policy.foo", "escalation_rule.0.escalation_delay_in_minutes", "10"),
					resource.TestCheckResourceAttr(
						"pagerduty_escalation_policy.foo", "escalation_rule.0.target.#", "1"),
					resource.TestCheckResourceAttr(
						"pagerduty_escalation_policy.foo", "escalation_rule.0.target.0.id", userID),
					resource.TestCheckResourceAttr(
						"pagerduty_escalation_policy.foo", "escalation_rule.0.target.0.type", "user"),
				),
			},
			resource.TestStep{
				Config: testAccCheckPagerDutyEscalationPolicyConfigUpdated(userID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyEscalationPolicyExists("pagerduty_escalation_policy.foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_escalation_policy.foo", "name", "bar"),
					resource.TestCheckResourceAttr(
						"pagerduty_escalation_policy.foo", "description", "bar"),
					resource.TestCheckResourceAttr(
						"pagerduty_escalation_policy.foo", "num_loops", "2"),
					resource.TestCheckResourceAttr(
						"pagerduty_escalation_policy.foo", "escalation_rule.#", "2"),
					resource.TestCheckResourceAttr(
						"pagerduty_escalation_policy.foo", "escalation_rule.0.escalation_delay_in_minutes", "10"),
					resource.TestCheckResourceAttr(
						"pagerduty_escalation_policy.foo", "escalation_rule.0.target.#", "1"),
					resource.TestCheckResourceAttr(
						"pagerduty_escalation_policy.foo", "escalation_rule.0.target.0.id", userID),
					resource.TestCheckResourceAttr(
						"pagerduty_escalation_policy.foo", "escalation_rule.0.target.0.type", "user"),
				),
			},
		},
	})
}

func testAccCheckPagerDutyEscalationPolicyDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*pagerduty.Client)
	for _, r := range s.RootModule().Resources {
		if r.Type != "pagerduty_escalation_policy" {
			continue
		}

		_, err := client.GetEscalationPolicy(r.Primary.ID, &pagerduty.GetEscalationPolicyOptions{})

		if err == nil {
			return fmt.Errorf("Escalation Policy still exists")
		}

	}
	return nil
}

func testAccCheckPagerDutyEscalationPolicyExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*pagerduty.Client)
		for _, r := range s.RootModule().Resources {
			if _, err := client.GetEscalationPolicy(r.Primary.ID, &pagerduty.GetEscalationPolicyOptions{}); err != nil {
				return fmt.Errorf("Received an error retrieving escalation_policy %s ID: %s", err, r.Primary.ID)
			}
		}
		return nil
	}
}

func testAccCheckPagerDutyEscalationPolicyConfig(id string) string {
	return fmt.Sprintf(`
resource "pagerduty_escalation_policy" "foo" {
  name        = "foo"
  description = "foo"
  num_loops   = 1

	escalation_rule {
	  escalation_delay_in_minutes = 10
		target {
		  type = "user"
			id = "%s"
		}
	}
}
	`, id)
}

func testAccCheckPagerDutyEscalationPolicyConfigUpdated(id string) string {
	return fmt.Sprintf(`
resource "pagerduty_escalation_policy" "foo" {
  name        = "bar"
  description = "bar"
  num_loops   = 2

	escalation_rule {
		escalation_delay_in_minutes = 10
		target {
			type = "user"
			id = "%[1]v"
		}
	}

	escalation_rule {
		escalation_delay_in_minutes = 20
		target {
			type = "user"
			id = "%[1]v"
		}
	}
}
`, userID)
}
