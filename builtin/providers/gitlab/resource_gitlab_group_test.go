package gitlab

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/xanzy/go-gitlab"
)

func TestAccGitlabGroup_basic(t *testing.T) {
	var group gitlab.Group
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckGitlabGroupDestroy,
		Steps: []resource.TestStep{
			// Create a group
			{
				Config: testAccGitlabGroupConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabGroupExists("gitlab_group.foo", &group),
					testAccCheckGitlabGroupAttributes(&group, &testAccGitlabGroupExpectedAttributes{
						Name:        fmt.Sprintf("foo-name-%d", rInt),
						Path:        fmt.Sprintf("foo-path-%d", rInt),
						Description: "Terraform acceptance tests",
						LFSEnabled:  true,
					}),
				),
			},
			// Update the group to change the description
			{
				Config: testAccGitlabGroupUpdateConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabGroupExists("gitlab_group.foo", &group),
					testAccCheckGitlabGroupAttributes(&group, &testAccGitlabGroupExpectedAttributes{
						Name:                 fmt.Sprintf("bar-name-%d", rInt),
						Path:                 fmt.Sprintf("bar-path-%d", rInt),
						Description:          "Terraform acceptance tests! Updated description",
						RequestAccessEnabled: true,
					}),
				),
			},
			// Update the group to put the anem and description back
			{
				Config: testAccGitlabGroupConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabGroupExists("gitlab_group.foo", &group),
					testAccCheckGitlabGroupAttributes(&group, &testAccGitlabGroupExpectedAttributes{
						Name:        fmt.Sprintf("foo-name-%d", rInt),
						Path:        fmt.Sprintf("foo-path-%d", rInt),
						Description: "Terraform acceptance tests",
						LFSEnabled:  true,
					}),
				),
			},
		},
	})
}

func testAccCheckGitlabGroupExists(n string, group *gitlab.Group) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not Found: %s", n)
		}

		groupID := rs.Primary.ID
		if groupID == "" {
			return fmt.Errorf("No group ID is set")
		}
		conn := testAccProvider.Meta().(*gitlab.Client)

		gotGroup, _, err := conn.Groups.GetGroup(groupID)
		if err != nil {
			return err
		}
		*group = *gotGroup
		return nil
	}
}

type testAccGitlabGroupExpectedAttributes struct {
	Name                 string
	Path                 string
	Description          string
	LFSEnabled           bool
	RequestAccessEnabled bool
}

func testAccCheckGitlabGroupAttributes(group *gitlab.Group, want *testAccGitlabGroupExpectedAttributes) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if group.Name != want.Name {
			return fmt.Errorf("got repo %q; want %q", group.Name, want.Name)
		}

		if group.Path != want.Path {
			return fmt.Errorf("got path %q; want %q", group.Path, want.Path)
		}

		if group.Description != want.Description {
			return fmt.Errorf("got description %q; want %q", group.Description, want.Description)
		}

		if group.LFSEnabled != want.LFSEnabled {
			return fmt.Errorf("got lfs_enabled %t; want %t", group.LFSEnabled, want.LFSEnabled)
		}

		if group.RequestAccessEnabled != want.RequestAccessEnabled {
			return fmt.Errorf("got request_access_enabled %t; want %t", group.RequestAccessEnabled, want.RequestAccessEnabled)
		}

		return nil
	}
}

func testAccCheckGitlabGroupDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*gitlab.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_group" {
			continue
		}

		group, resp, err := conn.Groups.GetGroup(rs.Primary.ID)
		if err == nil {
			if group != nil && fmt.Sprintf("%d", group.ID) == rs.Primary.ID {
				return fmt.Errorf("Group still exists")
			}
		}
		if resp.StatusCode != 404 {
			return err
		}
		return nil
	}
	return nil
}

func testAccGitlabGroupConfig(rInt int) string {
	return fmt.Sprintf(`
resource "gitlab_group" "foo" {
  name = "foo-name-%d"
  path = "foo-path-%d"
  description = "Terraform acceptance tests"

  # So that acceptance tests can be run in a gitlab organization
  # with no billing
  visibility_level = "public"
}
  `, rInt, rInt)
}

func testAccGitlabGroupUpdateConfig(rInt int) string {
	return fmt.Sprintf(`
resource "gitlab_group" "foo" {
  name = "bar-name-%d"
  path = "bar-path-%d"
  description = "Terraform acceptance tests! Updated description"
  lfs_enabled = false
  request_access_enabled = true

  # So that acceptance tests can be run in a gitlab organization
  # with no billing
  visibility_level = "public"
}
  `, rInt, rInt)
}
