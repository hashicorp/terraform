package github

import (
	"fmt"
	"os"
	"testing"

	"github.com/google/go-github/github"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccGithubRepositoryFork_basic(t *testing.T) {
	testForker := os.Getenv("GITHUB_TEST_FORK")
	testAccGithubRepositoryForkConfig := fmt.Sprintf(`
		resource "github_repository_fork" "test_repo_fork" {
			repository = "%s"
			username = "%s"
		}
	`, testRepo, testForker)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckGithubRepositoryForkDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccGithubRepositoryForkConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGithubRepositoryCollaboratorExists("github_repository_fork.test_repo_fork"),
					testAccCheckGithubRepositoryCollaboratorPermission("github_repository_collaborator.test_repo_collaborator"),
				),
			},
		},
	})
}

func testAccCheckGithubRepositoryForkDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*Clients).OrgClient

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "github_repository_fork" {
			continue
		}

		o := testAccProvider.Meta().(*Clients).OrgName
		r, _ := parseTwoPartID(rs.Primary.ID)
		repositories, _, err := conn.Repositories.ListForks(o, r, nil)
		if err != nil {
			return err
		}

		if !isIn(repositories, r) {
			return fmt.Errorf("Repository does not exists")
		}

		return nil
	}

	return nil
}

func isIn(repos []github.Repository, r string) bool {
	var b bool
	for _, repo := range repos {
		if *repo.Name == r {
			b = true
		}
	}

	return b
}

func testAccCheckGithubRepositoryForkExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		return nil
	}
}
