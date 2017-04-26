package github

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-github/github"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccGithubTeam_basic(t *testing.T) {
	var team github.Team
	randString := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	name := fmt.Sprintf("tf-acc-test-%s", randString)
	updatedName := fmt.Sprintf("tf-acc-test-updated-%s", randString)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckGithubTeamDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGithubTeamConfig(randString),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGithubTeamExists("github_team.foo", &team),
					testAccCheckGithubTeamAttributes(&team, name, "Terraform acc test group"),
				),
			},
			{
				Config: testAccGithubTeamUpdateConfig(randString),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGithubTeamExists("github_team.foo", &team),
					testAccCheckGithubTeamAttributes(&team, updatedName, "Terraform acc test group - updated"),
				),
			},
		},
	})
}

func TestAccGithubTeam_importBasic(t *testing.T) {
	randString := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckGithubTeamDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGithubTeamConfig(randString),
			},
			{
				ResourceName:      "github_team.foo",
				ImportState:       true,
				ImportStateVerify: true,
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
		githubTeam, _, err := conn.Organizations.GetTeam(context.TODO(), toGithubID(rs.Primary.ID))
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

		team, resp, err := conn.Organizations.GetTeam(context.TODO(), toGithubID(rs.Primary.ID))
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

func testAccGithubTeamConfig(randString string) string {
	return fmt.Sprintf(`
resource "github_team" "foo" {
	name = "tf-acc-test-%s"
	description = "Terraform acc test group"
	privacy = "secret"
}
`, randString)
}

func testAccGithubTeamUpdateConfig(randString string) string {
	return fmt.Sprintf(`
resource "github_team" "foo" {
	name = "tf-acc-test-updated-%s"
	description = "Terraform acc test group - updated"
	privacy = "closed"
}
`, randString)
}
