package bitbucket

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccBitbucketDefaultReviewers_basic(t *testing.T) {

	testUser := os.Getenv("BITBUCKET_USERNAME")
	testAccBitbucketDefaultReviewersConfig := fmt.Sprintf(`
		resource "bitbucket_repository" "test_repo" {
			owner = "%s"
			name = "test-repo-default-reviewers"
		}

		resource "bitbucket_default_reviewers" "test_reviewers" {
			owner = "%s"
			repository = "${bitbucket_repository.test_repo.name}"
			reviewers = [
				"%s",
			]
		}
	`, testUser, testUser, testUser)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckBitbucketDefaultReviewersDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccBitbucketDefaultReviewersConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBitbucketDefaultReviewersExists("bitbucket_default_reviewers.test_reviewers"),
				),
			},
		},
	})
}

func testAccCheckBitbucketDefaultReviewersDestroy(s *terraform.State) error {
	_, ok := s.RootModule().Resources["bitbucket_default_reviewers.test_reviewers"]
	if !ok {
		return fmt.Errorf("Not found %s", "bitbucket_default_reviewers.test_reviewers")
	}
	return nil
}

func testAccCheckBitbucketDefaultReviewersExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No default reviewers ID is set")
		}

		return nil
	}
}
