package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccSnsTopic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccAwsSnsTopicDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAwsSnsTopicConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccAwsSnsTopic(
						"aws_sns_topic.foo",
					),
				),
			},
			resource.TestStep{
				Config: testAccAwsSnsTopicConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccAwsSnsTopic(
						"aws_sns_topic.foo",
					),
				),
			},
		},
	})
}

func testAccAwsSnsTopicDestroy(s *terraform.State) error {
	if len(s.RootModule().Resources) > 0 {
		return fmt.Errorf("Expected all resources to be gone, but found: %#v", s.RootModule().Resources)
	}

	return nil
}

func testAccAwsSnsTopic(snsTopicResource string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[snsTopicResource]
		if !ok {
			return fmt.Errorf("Not found: %s", snsTopicResource)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}
		topic, ok := s.RootModule().Resources[snsTopicResource]
		if !ok {
			return fmt.Errorf("Not found: %s", snsTopicResource)
		}

		snsconn := testAccProvider.Meta().(*AWSClient).snsconn

		match, err := seekSnsTopic(topic.Primary.ID, snsconn)
		if err != nil {
			return err
		}
		if match == "" {
			return fmt.Errorf("Not found in AWS: %s", topic)
		}

		return nil
	}
}

const testAccAwsSnsTopicConfig = `
resource "aws_sns_topic" "foo" {
	name = "foo"
}
`

// Change the name but leave the resource name the same.
const testAccAwsSnsTopicConfigUpdate = `
resource "aws_sns_topic" "foo" {
	name = "bar"
}
`

func Test_parseSnsTopicArn(t *testing.T) {
	for _, ts := range []struct {
		arn    string
		wanted string
	}{
		{"arn:aws:sns:us-east-1:123456789012:foo", "foo"},
		{"arn:aws:sns:us-west-1:123456789012:bar", "bar"},
		{"arn:aws:sns:us-east-1:123456789012:baz", "baz"},
	} {
		got := parseSnsTopicArn(ts.arn)
		if got != ts.wanted {
			t.Fatalf("got %s; wanted %s", got, ts.wanted)
		}
	}
}
