package pagerduty

import (
	"fmt"
	"testing"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccPagerDutyUser_Basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPagerDutyUserDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckPagerDutyUserConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyUserExists("pagerduty_user.foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_user.foo", "name", "foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_user.foo", "email", "foo@bar.com"),
					resource.TestCheckResourceAttr(
						"pagerduty_user.foo", "color", "green"),
					resource.TestCheckResourceAttr(
						"pagerduty_user.foo", "role", "user"),
					resource.TestCheckResourceAttr(
						"pagerduty_user.foo", "job_title", "foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_user.foo", "description", "foo"),
				),
			},
			resource.TestStep{
				Config: testAccCheckPagerDutyUserConfigUpdated,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyUserExists("pagerduty_user.foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_user.foo", "name", "bar"),
					resource.TestCheckResourceAttr(
						"pagerduty_user.foo", "email", "bar@foo.com"),
					resource.TestCheckResourceAttr(
						"pagerduty_user.foo", "color", "red"),
					resource.TestCheckResourceAttr(
						"pagerduty_user.foo", "role", "user"),
					resource.TestCheckResourceAttr(
						"pagerduty_user.foo", "job_title", "bar"),
					resource.TestCheckResourceAttr(
						"pagerduty_user.foo", "description", "bar"),
				),
			},
		},
	})
}

func testAccCheckPagerDutyUserDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*pagerduty.Client)
	for _, r := range s.RootModule().Resources {
		if r.Type != "pagerduty_user" {
			continue
		}

		opts := pagerduty.GetUserOptions{}

		_, err := client.GetUser(r.Primary.ID, opts)

		if err == nil {
			return fmt.Errorf("User still exists")
		}

	}
	return nil
}

func testAccCheckPagerDutyUserExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*pagerduty.Client)
		for _, r := range s.RootModule().Resources {
			opts := pagerduty.GetUserOptions{}
			if _, err := client.GetUser(r.Primary.ID, opts); err != nil {
				return fmt.Errorf("Received an error retrieving user %s ID: %s", err, r.Primary.ID)
			}
		}
		return nil
	}
}

const testAccCheckPagerDutyUserConfig = `
resource "pagerduty_user" "foo" {
  name        = "foo"
  email       = "foo@bar.com"
  color       = "green"
  role        = "user"
  job_title   = "foo"
  description = "foo"
}
`

const testAccCheckPagerDutyUserConfigUpdated = `
resource "pagerduty_user" "foo" {
  name        = "bar"
  email       = "bar@foo.com"
  color       = "red"
  role        = "user"
  job_title   = "bar"
  description = "bar"
}
`
