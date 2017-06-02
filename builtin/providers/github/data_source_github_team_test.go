package github

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccGithubTeamDataSource_noMatchReturnsError(t *testing.T) {
	slug := "non-existing"
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      testAccCheckGithubTeamDataSourceConfig(slug),
				ExpectError: regexp.MustCompile(`Could not find team`),
			},
		},
	})
}

func testAccCheckGithubTeamDataSourceConfig(slug string) string {
	return fmt.Sprintf(`
data "github_team" "test" {
	slug = "%s"
}
`, slug)
}
