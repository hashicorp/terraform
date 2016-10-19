package gitlab

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/xanzy/go-gitlab"
)

func TestAccGitlabProject_basic(t *testing.T) {
	var project gitlab.Project

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckGitlabProjectDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccGitlabProjectConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectExists("gitlab_project.foo", &project),
					testAccCheckGitlabProjectAttributes(&project, &testAccGitlabProjectExpectedAttributes{
						Name:            "foo",
						Description:     "Terraform acceptance tests",
						VisibilityLevel: 20,
					}),
				),
			},
			resource.TestStep{
				Config: testAccGitlabProjectUpdateConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectExists("gitlab_project.foo", &project),
					testAccCheckGitlabProjectAttributes(&project, &testAccGitlabProjectExpectedAttributes{
						Name:            "foo",
						Description:     "Terraform acceptance tests!",
						VisibilityLevel: 20,
					}),
				),
			},
		},
	})
}

func testAccCheckGitlabProjectExists(n string, project *gitlab.Project) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not Found: %s", n)
		}

		repoName := rs.Primary.ID
		if repoName == "" {
			return fmt.Errorf("No project ID is set")
		}
		conn := testAccProvider.Meta().(*gitlab.Client)

		gotProject, _, err := conn.Projects.GetProject(repoName)
		if err != nil {
			return err
		}
		*project = *gotProject
		return nil
	}
}

type testAccGitlabProjectExpectedAttributes struct {
	Name            string
	Description     string
	VisibilityLevel gitlab.VisibilityLevelValue
}

func testAccCheckGitlabProjectAttributes(project *gitlab.Project, want *testAccGitlabProjectExpectedAttributes) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if project.Name != want.Name {
			return fmt.Errorf("got repo %q; want %q", project.Name, want.Name)
		}
		if project.Description != want.Description {
			return fmt.Errorf("got description %q; want %q", project.Description, want.Description)
		}

		if project.VisibilityLevel != want.VisibilityLevel {
			return fmt.Errorf("got default branch %q; want %q", project.VisibilityLevel, want.VisibilityLevel)
		}

		return nil
	}
}

func testAccCheckGitlabProjectDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*gitlab.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_project" {
			continue
		}

		gotRepo, resp, err := conn.Projects.GetProject(rs.Primary.ID)
		if err == nil {
			if gotRepo != nil && fmt.Sprintf("%d", gotRepo.ID) == rs.Primary.ID {
				return fmt.Errorf("Repository still exists")
			}
		}
		if resp.StatusCode != 404 {
			return err
		}
		return nil
	}
	return nil
}

const testAccGitlabProjectConfig = `
resource "gitlab_project" "foo" {
  name = "foo"
  description = "Terraform acceptance tests"

  # So that acceptance tests can be run in a github organization
  # with no billing
  visibility_level = "public"
}
`

const testAccGitlabProjectUpdateConfig = `
resource "gitlab_project" "foo" {
  name = "foo"
  description = "Terraform acceptance tests!"

  # So that acceptance tests can be run in a github organization
  # with no billing
  visibility_level = "public"
}
`
