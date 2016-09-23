package pagerduty

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccPagerDutyEscalationPolicy_import(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPagerDutyEscalationPolicyDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccPagerDutyEscalationPolicyConfigImported,
			},
			resource.TestStep{
				ResourceName:      "pagerduty_escalation_policy.foo",
				ImportState:       true,
				ImportStateVerify: false,
			},
		},
	})
}

const testAccPagerDutyEscalationPolicyConfigImported = `
resource "pagerduty_escalation_policy" "foo" {
  name = "foo"
	escalation_rule {
	  escalation_delay_in_minutes = 10
		target {
		  id = "PLBP04G"
			type = "user"
		}
	}
}
`
