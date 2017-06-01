package gitlab

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/xanzy/go-gitlab"
)

func TestAccGitlabProject_basic(t *testing.T) {
	var project gitlab.Project
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckGitlabProjectDestroy,
		Steps: []resource.TestStep{
			// Create a project with all the features on
			{
				Config: testAccGitlabProjectConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectExists("gitlab_project.foo", &project),
					testAccCheckGitlabProjectAttributes(&project, &testAccGitlabProjectExpectedAttributes{
						Name:                 fmt.Sprintf("foo-%d", rInt),
						Description:          "Terraform acceptance tests",
						IssuesEnabled:        true,
						MergeRequestsEnabled: true,
						WikiEnabled:          true,
						SnippetsEnabled:      true,
						VisibilityLevel:      20,
					}),
				),
			},
			// Update the project to turn the features off
			{
				Config: testAccGitlabProjectUpdateConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectExists("gitlab_project.foo", &project),
					testAccCheckGitlabProjectAttributes(&project, &testAccGitlabProjectExpectedAttributes{
						Name:            fmt.Sprintf("foo-%d", rInt),
						Description:     "Terraform acceptance tests!",
						VisibilityLevel: 20,
					}),
				),
			},
			//Update the project to turn the features on again
			{
				Config: testAccGitlabProjectConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectExists("gitlab_project.foo", &project),
					testAccCheckGitlabProjectAttributes(&project, &testAccGitlabProjectExpectedAttributes{
						Name:                 fmt.Sprintf("foo-%d", rInt),
						Description:          "Terraform acceptance tests",
						IssuesEnabled:        true,
						MergeRequestsEnabled: true,
						WikiEnabled:          true,
						SnippetsEnabled:      true,
						VisibilityLevel:      20,
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
	Name                 string
	Description          string
	DefaultBranch        string
	IssuesEnabled        bool
	MergeRequestsEnabled bool
	WikiEnabled          bool
	SnippetsEnabled      bool
	VisibilityLevel      gitlab.VisibilityLevelValue
}

func testAccCheckGitlabProjectAttributes(project *gitlab.Project, want *testAccGitlabProjectExpectedAttributes) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if project.Name != want.Name {
			return fmt.Errorf("got repo %q; want %q", project.Name, want.Name)
		}
		if project.Description != want.Description {
			return fmt.Errorf("got description %q; want %q", project.Description, want.Description)
		}

		if project.DefaultBranch != want.DefaultBranch {
			return fmt.Errorf("got default_branch %q; want %q", project.DefaultBranch, want.DefaultBranch)
		}

		if project.IssuesEnabled != want.IssuesEnabled {
			return fmt.Errorf("got issues_enabled %t; want %t", project.IssuesEnabled, want.IssuesEnabled)
		}

		if project.MergeRequestsEnabled != want.MergeRequestsEnabled {
			return fmt.Errorf("got merge_requests_enabled %t; want %t", project.MergeRequestsEnabled, want.MergeRequestsEnabled)
		}

		if project.WikiEnabled != want.WikiEnabled {
			return fmt.Errorf("got wiki_enabled %t; want %t", project.WikiEnabled, want.WikiEnabled)
		}

		if project.SnippetsEnabled != want.SnippetsEnabled {
			return fmt.Errorf("got snippets_enabled %t; want %t", project.SnippetsEnabled, want.SnippetsEnabled)
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

func testAccGitlabProjectConfig(rInt int) string {
	return fmt.Sprintf(`
resource "gitlab_project" "foo" {
  name = "foo-%d"
  description = "Terraform acceptance tests"

  # So that acceptance tests can be run in a gitlab organization
  # with no billing
  visibility_level = "public"
}
	`, rInt)
}

func testAccGitlabProjectUpdateConfig(rInt int) string {
	return fmt.Sprintf(`
resource "gitlab_project" "foo" {
  name = "foo-%d"
  description = "Terraform acceptance tests!"

  # So that acceptance tests can be run in a gitlab organization
  # with no billing
  visibility_level = "public"

  issues_enabled = false
  merge_requests_enabled = false
  wiki_enabled = false
  snippets_enabled = false
}
	`, rInt)
}
