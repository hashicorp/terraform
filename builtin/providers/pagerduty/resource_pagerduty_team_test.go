package pagerduty

import (
	"fmt"
	"testing"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccPagerDutyTeam_Basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPagerDutyTeamDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckPagerDutyTeamConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyTeamExists("pagerduty_team.foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_team.foo", "name", "foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_team.foo", "description", "foo"),
				),
			},
			resource.TestStep{
				Config: testAccCheckPagerDutyTeamConfigUpdated,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyTeamExists("pagerduty_team.foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_team.foo", "name", "bar"),
					resource.TestCheckResourceAttr(
						"pagerduty_team.foo", "description", "bar"),
				),
			},
		},
	})
}

func testAccCheckPagerDutyTeamDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*pagerduty.Client)
	for _, r := range s.RootModule().Resources {
		if r.Type != "pagerduty_team" {
			continue
		}

		_, err := client.GetTeam(r.Primary.ID)

		if err == nil {
			return fmt.Errorf("Team still exists")
		}

	}
	return nil
}

func testAccCheckPagerDutyTeamExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*pagerduty.Client)
		for _, r := range s.RootModule().Resources {
			if _, err := client.GetTeam(r.Primary.ID); err != nil {
				return fmt.Errorf("Received an error retrieving team %s ID: %s", err, r.Primary.ID)
			}
		}
		return nil
	}
}

const testAccCheckPagerDutyTeamConfig = `
resource "pagerduty_team" "foo" {
  name        = "foo"
  description = "foo"
}
`

const testAccCheckPagerDutyTeamConfigUpdated = `
resource "pagerduty_team" "foo" {
  name        = "bar"
  description = "bar"
}
`
