package github

import (
	"fmt"
	"testing"

	"github.com/google/go-github/github"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccGithubTeamRepository_basic(t *testing.T) {
	var repository github.Repository

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckGithubTeamRepositoryDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccGithubTeamRepositoryConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGithubTeamRepositoryExists("github_team_repository.test_team_test_repo", &repository),
					testAccCheckGithubTeamRepositoryRoleState("pull", &repository),
				),
			},
			resource.TestStep{
				Config: testAccGithubTeamRepositoryUpdateConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGithubTeamRepositoryExists("github_team_repository.test_team_test_repo", &repository),
					testAccCheckGithubTeamRepositoryRoleState("push", &repository),
				),
			},
		},
	})
}

func TestAccCheckGetPermissions(t *testing.T) {
	pullMap := map[string]bool{"pull": true, "push": false, "admin": false}
	pushMap := map[string]bool{"pull": true, "push": true, "admin": false}
	adminMap := map[string]bool{"pull": true, "push": true, "admin": true}
	errorMap := map[string]bool{"pull": false, "push": false, "admin": false}

	pull, _ := getRepoPermission(&pullMap)
	if pull != "pull" {
		t.Fatalf("Expected pull permission, actual: %s", pull)
	}

	push, _ := getRepoPermission(&pushMap)
	if push != "push" {
		t.Fatalf("Expected push permission, actual: %s", push)
	}

	admin, _ := getRepoPermission(&adminMap)
	if admin != "admin" {
		t.Fatalf("Expected admin permission, actual: %s", admin)
	}

	errPerm, err := getRepoPermission(&errorMap)
	if err == nil {
		t.Fatalf("Expected an error getting permissions, actual: %v", errPerm)
	}
}

func testAccCheckGithubTeamRepositoryRoleState(role string, repository *github.Repository) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceRole, err := getRepoPermission(repository.Permissions)
		if err != nil {
			return err
		}

		if resourceRole != role {
			return fmt.Errorf("Team repository role %v in resource does match expected state of %v", resourceRole, role)
		}
		return nil
	}
}

func testAccCheckGithubTeamRepositoryExists(n string, repository *github.Repository) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not Found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No team repository ID is set")
		}

		conn := testAccProvider.Meta().(*Organization).client
		t, r := parseTwoPartID(rs.Primary.ID)

		repo, _, err := conn.Organizations.IsTeamRepo(toGithubID(t),
			testAccProvider.Meta().(*Organization).name, r)

		if err != nil {
			return err
		}
		*repository = *repo
		return nil
	}
}

func testAccCheckGithubTeamRepositoryDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*Organization).client

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "github_team_repository" {
			continue
		}
		t, r := parseTwoPartID(rs.Primary.ID)

		repo, resp, err := conn.Organizations.IsTeamRepo(toGithubID(t),
			testAccProvider.Meta().(*Organization).name, r)

		if err == nil {
			if repo != nil &&
				buildTwoPartID(&t, repo.Name) == rs.Primary.ID {
				return fmt.Errorf("Team repository still exists")
			}
		}
		if resp.StatusCode != 404 {
			return err
		}
		return nil
	}
	return nil
}

const testAccGithubTeamRepositoryConfig = `
resource "github_team" "test_team" {
	name = "foo"
	description = "Terraform acc test group"
}

resource "github_team_repository" "test_team_test_repo" {
	team_id = "${github_team.test_team.id}"
	repository = "test-repo"
	permission = "pull"
}
`

const testAccGithubTeamRepositoryUpdateConfig = `
resource "github_team" "test_team" {
	name = "foo"
	description = "Terraform acc test group"
}

resource "github_team_repository" "test_team_test_repo" {
	team_id = "${github_team.test_team.id}"
	repository = "test-repo"
	permission = "push"
}
`
