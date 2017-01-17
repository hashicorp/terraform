package github

import (
	"fmt"
	"testing"

	"github.com/google/go-github/github"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

const expectedPermission string = "admin"

func TestAccGithubRepositoryCollaborator_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckGithubRepositoryCollaboratorDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccGithubRepositoryCollaboratorConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGithubRepositoryCollaboratorExists("github_repository_collaborator.test_repo_collaborator"),
					testAccCheckGithubRepositoryCollaboratorPermission("github_repository_collaborator.test_repo_collaborator"),
				),
			},
		},
	})
}

func TestAccGithubRepositoryCollaborator_importBasic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckGithubRepositoryCollaboratorDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccGithubRepositoryCollaboratorConfig,
			},
			resource.TestStep{
				ResourceName:      "github_repository_collaborator.test_repo_collaborator",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckGithubRepositoryCollaboratorDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*Organization).client

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "github_repository_collaborator" {
			continue
		}

		o := testAccProvider.Meta().(*Organization).name
		r, u := parseTwoPartID(rs.Primary.ID)
		isCollaborator, _, err := conn.Repositories.IsCollaborator(o, r, u)

		if err != nil {
			return err
		}

		if isCollaborator {
			return fmt.Errorf("Repository collaborator still exists")
		}

		return nil
	}

	return nil
}

func testAccCheckGithubRepositoryCollaboratorExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not Found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No membership ID is set")
		}

		conn := testAccProvider.Meta().(*Organization).client
		o := testAccProvider.Meta().(*Organization).name
		r, u := parseTwoPartID(rs.Primary.ID)

		isCollaborator, _, err := conn.Repositories.IsCollaborator(o, r, u)

		if err != nil {
			return err
		}

		if !isCollaborator {
			return fmt.Errorf("Repository collaborator does not exist")
		}

		return nil
	}
}

func testAccCheckGithubRepositoryCollaboratorPermission(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not Found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No membership ID is set")
		}

		conn := testAccProvider.Meta().(*Organization).client
		o := testAccProvider.Meta().(*Organization).name
		r, u := parseTwoPartID(rs.Primary.ID)

		collaborators, _, err := conn.Repositories.ListCollaborators(o, r, &github.ListOptions{})

		if err != nil {
			return err
		}

		for _, c := range collaborators {
			if *c.Login == u {
				permName, err := getRepoPermission(c.Permissions)

				if err != nil {
					return err
				}

				if permName != expectedPermission {
					return fmt.Errorf("Expected permission %s on repository collaborator, actual permission %s", expectedPermission, permName)
				}

				return nil
			}
		}

		return fmt.Errorf("Repository collaborator did not appear in list of collaborators on repository")
	}
}

var testAccGithubRepositoryCollaboratorConfig string = fmt.Sprintf(`
  resource "github_repository_collaborator" "test_repo_collaborator" {
    repository = "%s"
    username = "%s"
    permission = "%s"
  }
`, testRepo, testCollaborator, expectedPermission)
