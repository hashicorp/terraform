package aws

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAwsSnsTopicDataSource_noMatchReturnsError(t *testing.T) {
	name := "hashicorp.com"
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      testAccCheckAwsSnsTopicDataSourceConfig(name),
				ExpectError: regexp.MustCompile(`No topic with name domain`),
			},
			{
				Config:      testAccCheckAwsSnsTopicDataSourceConfigWithStatus(name),
				ExpectError: regexp.MustCompile(`No topic with name`),
			},
		},
	})
}

func testAccCheckAwsSnsTopicDataSourceConfig(name string) string {
	return fmt.Sprintf(`
data "aws_sns_topic" "test" {
	name = "%s"
}
`, name)
}

func testAccCheckAwsSnsTopicDataSourceConfigWithStatus(name string) string {
	return fmt.Sprintf(`
data "aws_sns_topic" "test" {
	name = "%s"
}
`, "")
}
