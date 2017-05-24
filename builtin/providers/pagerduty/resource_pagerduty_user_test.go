package pagerduty

import (
	"fmt"
	"testing"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccPagerDutyUser_Basic(t *testing.T) {
	username := fmt.Sprintf("tf-%s", acctest.RandString(5))
	usernameUpdated := fmt.Sprintf("tf-%s", acctest.RandString(5))
	email := fmt.Sprintf("%s@foo.com", username)
	emailUpdated := fmt.Sprintf("%s@foo.com", usernameUpdated)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPagerDutyUserDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckPagerDutyUserConfig(username, email),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyUserExists("pagerduty_user.foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_user.foo", "name", username),
					resource.TestCheckResourceAttr(
						"pagerduty_user.foo", "email", email),
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
			{
				Config: testAccCheckPagerDutyUserConfigUpdated(usernameUpdated, emailUpdated),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyUserExists("pagerduty_user.foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_user.foo", "name", usernameUpdated),
					resource.TestCheckResourceAttr(
						"pagerduty_user.foo", "email", emailUpdated),
					resource.TestCheckResourceAttr(
						"pagerduty_user.foo", "color", "red"),
					resource.TestCheckResourceAttr(
						"pagerduty_user.foo", "role", "team_responder"),
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
	username := fmt.Sprintf("tf-%s", acctest.RandString(5))
	email := fmt.Sprintf("%s@foo.com", username)
	team1 := fmt.Sprintf("tf-%s", acctest.RandString(5))
	team2 := fmt.Sprintf("tf-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPagerDutyUserDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckPagerDutyUserWithTeamsConfig(team1, username, email),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyUserExists("pagerduty_user.foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_user.foo", "name", username),
					resource.TestCheckResourceAttr(
						"pagerduty_user.foo", "email", email),
					resource.TestCheckResourceAttr(
						"pagerduty_user.foo", "teams.#", "1"),
				),
			},
			{
				Config: testAccCheckPagerDutyUserWithTeamsConfigUpdated(team1, team2, username, email),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyUserExists("pagerduty_user.foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_user.foo", "name", username),
					resource.TestCheckResourceAttr(
						"pagerduty_user.foo", "email", email),
					resource.TestCheckResourceAttr(
						"pagerduty_user.foo", "teams.#", "2"),
				),
			},
			{
				Config: testAccCheckPagerDutyUserWithNoTeamsConfig(team1, team2, username, email),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyUserExists("pagerduty_user.foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_user.foo", "name", username),
					resource.TestCheckResourceAttr(
						"pagerduty_user.foo", "email", email),
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

func testAccCheckPagerDutyUserConfig(username, email string) string {
	return fmt.Sprintf(`
resource "pagerduty_user" "foo" {
  name        = "%s"
  email       = "%s"
  color       = "green"
  role        = "user"
  job_title   = "foo"
  description = "foo"
}`, username, email)
}

func testAccCheckPagerDutyUserConfigUpdated(username, email string) string {
	return fmt.Sprintf(`
resource "pagerduty_user" "foo" {
  name        = "%s"
  email       = "%s"
  color       = "red"
  role        = "team_responder"
  job_title   = "bar"
  description = "bar"
}`, username, email)
}

func testAccCheckPagerDutyUserWithTeamsConfig(team, username, email string) string {
	return fmt.Sprintf(`
resource "pagerduty_team" "foo" {
  name = "%s"
}

resource "pagerduty_user" "foo" {
  name  = "%s"
  email = "%s"
  teams = ["${pagerduty_team.foo.id}"]
}
`, team, username, email)
}

func testAccCheckPagerDutyUserWithTeamsConfigUpdated(team1, team2, username, email string) string {
	return fmt.Sprintf(`
resource "pagerduty_team" "foo" {
  name = "%s"
}

resource "pagerduty_team" "bar" {
  name = "%s"
}

resource "pagerduty_user" "foo" {
  name  = "%s"
  email = "%s"
  teams = ["${pagerduty_team.foo.id}", "${pagerduty_team.bar.id}"]
}
`, team1, team2, username, email)
}

func testAccCheckPagerDutyUserWithNoTeamsConfig(team1, team2, username, email string) string {
	return fmt.Sprintf(`
resource "pagerduty_team" "foo" {
  name = "%s"
}

resource "pagerduty_team" "bar" {
  name = "%s"
}

resource "pagerduty_user" "foo" {
  name  = "%s"
  email = "%s"
}
`, team1, team2, username, email)
}
