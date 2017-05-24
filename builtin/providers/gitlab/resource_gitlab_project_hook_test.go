package gitlab

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/xanzy/go-gitlab"
)

func TestAccGitlabProjectHook_basic(t *testing.T) {
	var hook gitlab.ProjectHook
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckGitlabProjectHookDestroy,
		Steps: []resource.TestStep{
			// Create a project and hook with default options
			{
				Config: testAccGitlabProjectHookConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectHookExists("gitlab_project_hook.foo", &hook),
					testAccCheckGitlabProjectHookAttributes(&hook, &testAccGitlabProjectHookExpectedAttributes{
						URL:                   fmt.Sprintf("https://example.com/hook-%d", rInt),
						PushEvents:            true,
						EnableSSLVerification: true,
					}),
				),
			},
			// Update the project hook to toggle all the values to their inverse
			{
				Config: testAccGitlabProjectHookUpdateConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectHookExists("gitlab_project_hook.foo", &hook),
					testAccCheckGitlabProjectHookAttributes(&hook, &testAccGitlabProjectHookExpectedAttributes{
						URL:                   fmt.Sprintf("https://example.com/hook-%d", rInt),
						PushEvents:            false,
						IssuesEvents:          true,
						MergeRequestsEvents:   true,
						TagPushEvents:         true,
						NoteEvents:            true,
						BuildEvents:           true,
						PipelineEvents:        true,
						WikiPageEvents:        true,
						EnableSSLVerification: false,
					}),
				),
			},
			// Update the project hook to toggle the options back
			{
				Config: testAccGitlabProjectHookConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectHookExists("gitlab_project_hook.foo", &hook),
					testAccCheckGitlabProjectHookAttributes(&hook, &testAccGitlabProjectHookExpectedAttributes{
						URL:                   fmt.Sprintf("https://example.com/hook-%d", rInt),
						PushEvents:            true,
						EnableSSLVerification: true,
					}),
				),
			},
		},
	})
}

func testAccCheckGitlabProjectHookExists(n string, hook *gitlab.ProjectHook) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not Found: %s", n)
		}

		hookID, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return err
		}
		repoName := rs.Primary.Attributes["project"]
		if repoName == "" {
			return fmt.Errorf("No project ID is set")
		}
		conn := testAccProvider.Meta().(*gitlab.Client)

		gotHook, _, err := conn.Projects.GetProjectHook(repoName, hookID)
		if err != nil {
			return err
		}
		*hook = *gotHook
		return nil
	}
}

type testAccGitlabProjectHookExpectedAttributes struct {
	URL                   string
	PushEvents            bool
	IssuesEvents          bool
	MergeRequestsEvents   bool
	TagPushEvents         bool
	NoteEvents            bool
	BuildEvents           bool
	PipelineEvents        bool
	WikiPageEvents        bool
	EnableSSLVerification bool
}

func testAccCheckGitlabProjectHookAttributes(hook *gitlab.ProjectHook, want *testAccGitlabProjectHookExpectedAttributes) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if hook.URL != want.URL {
			return fmt.Errorf("got url %q; want %q", hook.URL, want.URL)
		}

		if hook.EnableSSLVerification != want.EnableSSLVerification {
			return fmt.Errorf("got enable_ssl_verification %t; want %t", hook.EnableSSLVerification, want.EnableSSLVerification)
		}

		if hook.PushEvents != want.PushEvents {
			return fmt.Errorf("got push_events %t; want %t", hook.PushEvents, want.PushEvents)
		}

		if hook.IssuesEvents != want.IssuesEvents {
			return fmt.Errorf("got issues_events %t; want %t", hook.IssuesEvents, want.IssuesEvents)
		}

		if hook.MergeRequestsEvents != want.MergeRequestsEvents {
			return fmt.Errorf("got merge_requests_events %t; want %t", hook.MergeRequestsEvents, want.MergeRequestsEvents)
		}

		if hook.TagPushEvents != want.TagPushEvents {
			return fmt.Errorf("got tag_push_events %t; want %t", hook.TagPushEvents, want.TagPushEvents)
		}

		if hook.NoteEvents != want.NoteEvents {
			return fmt.Errorf("got note_events %t; want %t", hook.NoteEvents, want.NoteEvents)
		}

		if hook.BuildEvents != want.BuildEvents {
			return fmt.Errorf("got build_events %t; want %t", hook.BuildEvents, want.BuildEvents)
		}

		if hook.PipelineEvents != want.PipelineEvents {
			return fmt.Errorf("got pipeline_events %t; want %t", hook.PipelineEvents, want.PipelineEvents)
		}

		if hook.WikiPageEvents != want.WikiPageEvents {
			return fmt.Errorf("got wiki_page_events %t; want %t", hook.WikiPageEvents, want.WikiPageEvents)
		}

		return nil
	}
}

func testAccCheckGitlabProjectHookDestroy(s *terraform.State) error {
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

func testAccGitlabProjectHookConfig(rInt int) string {
	return fmt.Sprintf(`
resource "gitlab_project" "foo" {
  name = "foo-%d"
  description = "Terraform acceptance tests"

  # So that acceptance tests can be run in a gitlab organization
  # with no billing
  visibility_level = "public"
}

resource "gitlab_project_hook" "foo" {
  project = "${gitlab_project.foo.id}"
  url = "https://example.com/hook-%d"
}
	`, rInt, rInt)
}

func testAccGitlabProjectHookUpdateConfig(rInt int) string {
	return fmt.Sprintf(`
resource "gitlab_project" "foo" {
  name = "foo-%d"
  description = "Terraform acceptance tests"

  # So that acceptance tests can be run in a gitlab organization
  # with no billing
  visibility_level = "public"
}

resource "gitlab_project_hook" "foo" {
  project = "${gitlab_project.foo.id}"
  url = "https://example.com/hook-%d"
  enable_ssl_verification = false
  push_events = false
  issues_events = true
  merge_requests_events = true
  tag_push_events = true
  note_events = true
  build_events = true
  pipeline_events = true
  wiki_page_events = true
}
	`, rInt, rInt)
}
