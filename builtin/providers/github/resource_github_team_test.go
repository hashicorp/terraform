package github

import (
	"fmt"
	"testing"

	"github.com/google/go-github/github"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccGithubTeam_basic(t *testing.T) {
	var team github.Team

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckGithubTeamDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccGithubTeamConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGithubTeamExists("github_team.foo", &team),
					testAccCheckGithubTeamAttributes(&team, "foo", "Terraform acc test group"),
				),
			},
			resource.TestStep{
				Config: testAccGithubTeamUpdateConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGithubTeamExists("github_team.foo", &team),
					testAccCheckGithubTeamAttributes(&team, "foo2", "Terraform acc test group - updated"),
				),
			},
		},
	})
}

func testAccCheckGithubTeamExists(n string, team *github.Team) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not Found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Team ID is set")
		}

		conn := testAccProvider.Meta().(*Organization).client
		githubTeam, _, err := conn.Organizations.GetTeam(toGithubID(rs.Primary.ID))
		if err != nil {
			return err
		}
		*team = *githubTeam
		return nil
	}
}

func testAccCheckGithubTeamAttributes(team *github.Team, name, description string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if *team.Name != name {
			return fmt.Errorf("Team name does not match: %s, %s", *team.Name, name)
		}

		if *team.Description != description {
			return fmt.Errorf("Team description does not match: %s, %s", *team.Description, description)
		}

		return nil
	}
}

func testAccCheckGithubTeamDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*Organization).client

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "github_team" {
			continue
		}

		team, resp, err := conn.Organizations.GetTeam(toGithubID(rs.Primary.ID))
		if err == nil {
			if team != nil &&
				fromGithubID(team.ID) == rs.Primary.ID {
				return fmt.Errorf("Team still exists")
			}
		}
		if resp.StatusCode != 404 {
			return err
		}
		return nil
	}
	return nil
}

const testAccGithubTeamConfig = `
resource "github_team" "foo" {
	name = "foo"
	description = "Terraform acc test group"
	privacy = "secret"
}
`

const testAccGithubTeamUpdateConfig = `
resource "github_team" "foo" {
	name = "foo2"
	description = "Terraform acc test group - updated"
	privacy = "closed"
}
`
