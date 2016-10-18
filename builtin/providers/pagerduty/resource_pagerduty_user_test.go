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

func TestAccPagerDutyUserWithTeams_Basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPagerDutyUserDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckPagerDutyUserWithTeamsConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyUserExists("pagerduty_user.foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_user.foo", "name", "foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_user.foo", "email", "foo@bar.com"),
					resource.TestCheckResourceAttr(
						"pagerduty_user.foo", "teams.#", "1"),
				),
			},
			resource.TestStep{
				Config: testAccCheckPagerDutyUserWithTeamsConfigUpdated,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyUserExists("pagerduty_user.foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_user.foo", "name", "foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_user.foo", "email", "foo@bar.com"),
					resource.TestCheckResourceAttr(
						"pagerduty_user.foo", "teams.#", "2"),
				),
			},
			resource.TestStep{
				Config: testAccCheckPagerDutyUserWithNoTeamsConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyUserExists("pagerduty_user.foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_user.foo", "name", "foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_user.foo", "email", "foo@bar.com"),
					resource.TestCheckResourceAttr(
						"pagerduty_user.foo", "teams.#", "0"),
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
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No user ID is set")
		}

		client := testAccProvider.Meta().(*pagerduty.Client)

		found, err := client.GetUser(rs.Primary.ID, pagerduty.GetUserOptions{})
		if err != nil {
			return err
		}

		if found.ID != rs.Primary.ID {
			return fmt.Errorf("User not found: %v - %v", rs.Primary.ID, found)
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

const testAccCheckPagerDutyUserWithTeamsConfig = `
resource "pagerduty_team" "foo" {
  name = "Foo team"
}

resource "pagerduty_user" "foo" {
  name  = "foo"
  email = "foo@bar.com"
  teams = ["${pagerduty_team.foo.id}"]
}
`
const testAccCheckPagerDutyUserWithTeamsConfigUpdated = `
resource "pagerduty_team" "foo" {
  name = "Foo team"
}

resource "pagerduty_team" "bar" {
  name = "Bar team"
}

resource "pagerduty_user" "foo" {
  name  = "foo"
  email = "foo@bar.com"
  teams = ["${pagerduty_team.foo.id}", "${pagerduty_team.bar.id}"]
}
`

const testAccCheckPagerDutyUserWithNoTeamsConfig = `
resource "pagerduty_team" "foo" {
  name = "Foo team"
}

resource "pagerduty_team" "bar" {
  name = "Bar team"
}

resource "pagerduty_user" "foo" {
  name  = "foo"
  email = "foo@bar.com"
}
`
