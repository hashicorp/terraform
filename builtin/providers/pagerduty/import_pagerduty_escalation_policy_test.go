package pagerduty

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccPagerDutyEscalationPolicy_import(t *testing.T) {
	username := fmt.Sprintf("tf-%s", acctest.RandString(5))
	email := fmt.Sprintf("%s@foo.com", username)
	escalationPolicy := fmt.Sprintf("tf-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPagerDutyEscalationPolicyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckPagerDutyEscalationPolicyConfig(username, email, escalationPolicy),
			},

			{
				ResourceName:      "pagerduty_escalation_policy.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
