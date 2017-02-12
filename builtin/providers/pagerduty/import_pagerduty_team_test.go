package pagerduty

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccPagerDutyTeam_import(t *testing.T) {
	resourceName := "pagerduty_team.foo"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPagerDutyTeamDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckPagerDutyTeamConfig,
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
