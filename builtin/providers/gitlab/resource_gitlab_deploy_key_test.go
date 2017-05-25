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

func TestAccGitlabDeployKey_basic(t *testing.T) {
	var deployKey gitlab.DeployKey
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckGitlabDeployKeyDestroy,
		Steps: []resource.TestStep{
			// Create a project and deployKey with default options
			{
				Config: testAccGitlabDeployKeyConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabDeployKeyExists("gitlab_deploy_key.foo", &deployKey),
					testAccCheckGitlabDeployKeyAttributes(&deployKey, &testAccGitlabDeployKeyExpectedAttributes{
						Title: fmt.Sprintf("deployKey-%d", rInt),
						Key:   "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCj13ozEBZ0s4el4k6mYqoyIKKKMh9hHY0sAYqSPXs2zGuVFZss1P8TPuwmdXVjHR7TiRXwC49zDrkyWJgiufggYJ1VilOohcMOODwZEJz+E5q4GCfHuh90UEh0nl8B2R0Uoy0LPeg93uZzy0hlHApsxRf/XZJz/1ytkZvCtxdllxfImCVxJReMeRVEqFCTCvy3YuJn0bce7ulcTFRvtgWOpQsr6GDK8YkcCCv2eZthVlrEwy6DEpAKTRiRLGgUj4dPO0MmO4cE2qD4ualY01PhNORJ8Q++I+EtkGt/VALkecwFuBkl18/gy+yxNJHpKc/8WVVinDeFrd/HhiY9yU0d richardc@tamborine.example.1",
					}),
				),
			},
			// Update the project deployKey to toggle all the values to their inverse
			{
				Config: testAccGitlabDeployKeyUpdateConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabDeployKeyExists("gitlab_deploy_key.foo", &deployKey),
					testAccCheckGitlabDeployKeyAttributes(&deployKey, &testAccGitlabDeployKeyExpectedAttributes{
						Title: fmt.Sprintf("modifiedDeployKey-%d", rInt),
						Key:   "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC6pSke2kb7YBjo65xDKegbOQsAtnMupRcFxXji7L1iXivGwORq0qpC2xzbhez5jk1WgPckEaNv2/Bz0uEW6oSIXw1KT1VN2WzEUfQCbpNyZPtn4iV3nyl6VQW/Nd1SrxiFJtH1H4vu+eCo4McMXTjuBBD06fiJNrHaSw734LjQgqtXWJuVym9qS5MqraZB7wDwTQwSM6kslL7KTgmo3ONsTLdb2zZhv6CS+dcFKinQo7/ttTmeMuXGbPOVuNfT/bePVIN1MF1TislHa2L2dZdGeoynNJT4fVPjA2Xl6eHWh4ySbvnfPznASsjBhP0n/QKprYJ/5fQShdBYBcuQiIMd richardc@tamborine.example.2",
					}),
				),
			},
			// Update the project deployKey to toggle the options back
			{
				Config: testAccGitlabDeployKeyConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabDeployKeyExists("gitlab_deploy_key.foo", &deployKey),
					testAccCheckGitlabDeployKeyAttributes(&deployKey, &testAccGitlabDeployKeyExpectedAttributes{
						Title: fmt.Sprintf("deployKey-%d", rInt),
						Key:   "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCj13ozEBZ0s4el4k6mYqoyIKKKMh9hHY0sAYqSPXs2zGuVFZss1P8TPuwmdXVjHR7TiRXwC49zDrkyWJgiufggYJ1VilOohcMOODwZEJz+E5q4GCfHuh90UEh0nl8B2R0Uoy0LPeg93uZzy0hlHApsxRf/XZJz/1ytkZvCtxdllxfImCVxJReMeRVEqFCTCvy3YuJn0bce7ulcTFRvtgWOpQsr6GDK8YkcCCv2eZthVlrEwy6DEpAKTRiRLGgUj4dPO0MmO4cE2qD4ualY01PhNORJ8Q++I+EtkGt/VALkecwFuBkl18/gy+yxNJHpKc/8WVVinDeFrd/HhiY9yU0d richardc@tamborine.example.1",
					}),
				),
			},
		},
	})
}

func testAccCheckGitlabDeployKeyExists(n string, deployKey *gitlab.DeployKey) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not Found: %s", n)
		}

		deployKeyID, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return err
		}
		repoName := rs.Primary.Attributes["project"]
		if repoName == "" {
			return fmt.Errorf("No project ID is set")
		}
		conn := testAccProvider.Meta().(*gitlab.Client)

		gotDeployKey, _, err := conn.DeployKeys.GetDeployKey(repoName, deployKeyID)
		if err != nil {
			return err
		}
		*deployKey = *gotDeployKey
		return nil
	}
}

type testAccGitlabDeployKeyExpectedAttributes struct {
	Title   string
	Key     string
	CanPush bool
}

func testAccCheckGitlabDeployKeyAttributes(deployKey *gitlab.DeployKey, want *testAccGitlabDeployKeyExpectedAttributes) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if deployKey.Title != want.Title {
			return fmt.Errorf("got title %q; want %q", deployKey.Title, want.Title)
		}

		if deployKey.Key != want.Key {
			return fmt.Errorf("got key %q; want %q", deployKey.Key, want.Key)
		}

		if deployKey.CanPush != nil && *deployKey.CanPush != want.CanPush {
			return fmt.Errorf("got can_push %t; want %t", *deployKey.CanPush, want.CanPush)
		}

		return nil
	}
}

func testAccCheckGitlabDeployKeyDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*gitlab.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_project" {
			continue
		}
		deployKeyID, err := strconv.Atoi(rs.Primary.ID)
		project := rs.Primary.Attributes["project"]

		gotDeployKey, resp, err := conn.DeployKeys.GetDeployKey(project, deployKeyID)
		if err == nil {
			if gotDeployKey != nil && fmt.Sprintf("%d", gotDeployKey.ID) == rs.Primary.ID {
				return fmt.Errorf("Deploy key still exists")
			}
		}
		if resp.StatusCode != 404 {
			return err
		}
		return nil
	}
	return nil
}

func testAccGitlabDeployKeyConfig(rInt int) string {
	return fmt.Sprintf(`
resource "gitlab_project" "foo" {
  name = "foo-%d"
  description = "Terraform acceptance tests"

  # So that acceptance tests can be run in a gitlab organization
  # with no billing
  visibility_level = "public"
}

resource "gitlab_deploy_key" "foo" {
  project = "${gitlab_project.foo.id}"
  title = "deployKey-%d"
  key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCj13ozEBZ0s4el4k6mYqoyIKKKMh9hHY0sAYqSPXs2zGuVFZss1P8TPuwmdXVjHR7TiRXwC49zDrkyWJgiufggYJ1VilOohcMOODwZEJz+E5q4GCfHuh90UEh0nl8B2R0Uoy0LPeg93uZzy0hlHApsxRf/XZJz/1ytkZvCtxdllxfImCVxJReMeRVEqFCTCvy3YuJn0bce7ulcTFRvtgWOpQsr6GDK8YkcCCv2eZthVlrEwy6DEpAKTRiRLGgUj4dPO0MmO4cE2qD4ualY01PhNORJ8Q++I+EtkGt/VALkecwFuBkl18/gy+yxNJHpKc/8WVVinDeFrd/HhiY9yU0d richardc@tamborine.example.1"
}
  `, rInt, rInt)
}

func testAccGitlabDeployKeyUpdateConfig(rInt int) string {
	return fmt.Sprintf(`
resource "gitlab_project" "foo" {
  name = "foo-%d"
  description = "Terraform acceptance tests"

  # So that acceptance tests can be run in a gitlab organization
  # with no billing
  visibility_level = "public"
}

resource "gitlab_deploy_key" "foo" {
  project = "${gitlab_project.foo.id}"
  title = "modifiedDeployKey-%d"
  key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC6pSke2kb7YBjo65xDKegbOQsAtnMupRcFxXji7L1iXivGwORq0qpC2xzbhez5jk1WgPckEaNv2/Bz0uEW6oSIXw1KT1VN2WzEUfQCbpNyZPtn4iV3nyl6VQW/Nd1SrxiFJtH1H4vu+eCo4McMXTjuBBD06fiJNrHaSw734LjQgqtXWJuVym9qS5MqraZB7wDwTQwSM6kslL7KTgmo3ONsTLdb2zZhv6CS+dcFKinQo7/ttTmeMuXGbPOVuNfT/bePVIN1MF1TislHa2L2dZdGeoynNJT4fVPjA2Xl6eHWh4ySbvnfPznASsjBhP0n/QKprYJ/5fQShdBYBcuQiIMd richardc@tamborine.example.2"
}
  `, rInt, rInt)
}
