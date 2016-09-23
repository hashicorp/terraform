package pagerduty

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccPagerDutyTeam_import(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPagerDutyTeamDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccPagerDutyTeamConfigImported,
			},
			resource.TestStep{
				ResourceName:      "pagerduty_team.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

const testAccPagerDutyTeamConfigImported = `
resource "pagerduty_team" "foo" {
  name = "foo"
	description = "foo"
}
`
