package bitbucket

import (
	"fmt"
	"net/url"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccBitbucketHook_basic(t *testing.T) {
	var hook Hook

	testUser := os.Getenv("BITBUCKET_USERNAME")
	testAccBitbucketHookConfig := fmt.Sprintf(`
		resource "bitbucket_repository" "test_repo" {
			owner = "%s"
			name = "test-repo-for-webhook-test"
		}
		resource "bitbucket_hook" "test_repo_hook" {
			owner = "%s"
			repository = "${bitbucket_repository.test_repo.name}"
			description = "Test hook for terraform"
			url = "https://httpbin.org"
			events = [
				"repo:push",
			]
		}
	`, testUser, testUser)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckBitbucketHookDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccBitbucketHookConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBitbucketHookExists("bitbucket_hook.test_repo_hook", &hook),
				),
			},
		},
	})
}

func testAccCheckBitbucketHookDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*BitbucketClient)
	rs, ok := s.RootModule().Resources["bitbucket_hook.test_repo_hook"]
	if !ok {
		return fmt.Errorf("Not found %s", "bitbucket_hook.test_repo_hook")
	}

	response, err := client.Get(fmt.Sprintf("2.0/repositories/%s/%s/hooks/%s", rs.Primary.Attributes["owner"], rs.Primary.Attributes["repository"], url.PathEscape(rs.Primary.Attributes["uuid"])))

	if err == nil {
		return fmt.Errorf("The resource was found should have errored")
	}

	if response.StatusCode != 404 {
		return fmt.Errorf("Hook still exists")
	}

	return nil
}

func testAccCheckBitbucketHookExists(n string, hook *Hook) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found %s", n)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No Hook ID is set")
		}
		return nil
	}
}
