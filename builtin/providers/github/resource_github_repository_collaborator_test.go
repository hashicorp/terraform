package github

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

const expectedPermission string = "admin"

func TestAccGithubRepositoryCollaborator_basic(t *testing.T) {
	repoName := fmt.Sprintf("tf-acc-test-collab-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckGithubRepositoryCollaboratorDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGithubRepositoryCollaboratorConfig(repoName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGithubRepositoryCollaboratorExists("github_repository_collaborator.test_repo_collaborator"),
					testAccCheckGithubRepositoryCollaboratorPermission("github_repository_collaborator.test_repo_collaborator"),
				),
			},
		},
	})
}

func TestAccGithubRepositoryCollaborator_importBasic(t *testing.T) {
	repoName := fmt.Sprintf("tf-acc-test-collab-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckGithubRepositoryCollaboratorDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGithubRepositoryCollaboratorConfig(repoName),
			},
			{
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
		isCollaborator, _, err := conn.Repositories.IsCollaborator(context.TODO(), o, r, u)

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

		invitations, _, err := conn.Repositories.ListInvitations(context.TODO(), o, r, nil)
		if err != nil {
			return err
		}

		hasInvitation := false
		for _, i := range invitations {
			if *i.Invitee.Login == u {
				hasInvitation = true
				break
			}
		}

		if !hasInvitation {
			return fmt.Errorf("Repository collaboration invitation does not exist")
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

		invitations, _, err := conn.Repositories.ListInvitations(context.TODO(), o, r, nil)
		if err != nil {
			return err
		}

		for _, i := range invitations {
			if *i.Invitee.Login == u {
				permName, err := getInvitationPermission(i)

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

func testAccGithubRepositoryCollaboratorConfig(repoName string) string {
	return fmt.Sprintf(`
resource "github_repository" "test" {
	name = "%s"
}

  resource "github_repository_collaborator" "test_repo_collaborator" {
    repository = "${github_repository.test.name}"
    username = "%s"
    permission = "%s"
  }
`, repoName, testCollaborator, expectedPermission)
}
