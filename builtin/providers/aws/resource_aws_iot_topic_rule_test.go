package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/service/iot"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSIoTTopicRule_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSIoTTopicRuleDestroy_basic,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSIoTTopicRule_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSIoTTopicRuleExists_basic("aws_iot_topic_rule.rule"),
				),
			},
		},
	})
}

func testAccCheckAWSIoTTopicRuleDestroy_basic(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).iotconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_iot_topic_rule" {
			continue
		}

		out, err := conn.ListTopicRules(&iot.ListTopicRulesInput{})

		if err != nil {
			return err
		}

		for _, r := range out.Rules {
			if *r.RuleName == rs.Primary.ID {
				return fmt.Errorf("IoT topic rule still exists:\n%s", r)
			}
		}

	}

	return nil
}

func testAccCheckAWSIoTTopicRuleExists_basic(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		return nil
	}
}

var testAccAWSIoTTopicRule_basic = `
resource "aws_iot_topic_rule" "rule" {
  name = "MyRule"
  description = "Example rule"
  enabled = true
  sql = "SELECT * FROM 'topic/test'"
  sql_version = "2015-10-08"

  // Fake data
  dynamodb {
    hash_key_field = "hash_key_field"
    hash_key_value = "hash_key_value"
    payload_field = "payload_field"
    range_key_field = "range_key_field"
    range_key_value = "range_key_value"
    role_arn = "${aws_iam_role.iot_role.arn}"
    table_name = "table_name"
  }

}

resource "aws_iam_role" "iot_role" {
    name = "iot"
    assume_role_policy = <<EOF
{
    "Version":"2012-10-17",
    "Statement":[{
        "Effect": "Allow",
        "Principal": {
            "Service": "iot.amazonaws.com"
        },
        "Action": "sts:AssumeRole"
    }]
}
EOF
}

resource "aws_iam_policy" "policy" {
    name = "test_policy"
    path = "/"
    description = "My test policy"
    policy = <<EOF
{
    "Version": "2012-10-17",
    "Statement": [{
        "Effect": "Allow",
        "Action": "dynamodb:*",
        "Resource": "*"
    }]
}
EOF
}

resource "aws_iam_policy_attachment" "attach_policy" {
    name = "attach_policy"
    roles = ["${aws_iam_role.iot_role.name}"]
    policy_arn = "${aws_iam_policy.policy.arn}"
}
`
