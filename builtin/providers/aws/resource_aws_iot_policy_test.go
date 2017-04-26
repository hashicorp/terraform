package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/service/iot"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSIoTPolicy_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSIoTPolicyDestroy_basic,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSIoTPolicy_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSIoTPolicyExists_basic("aws_iot_policy.pubsub"),
				),
			},
		},
	})
}

func testAccCheckAWSIoTPolicyDestroy_basic(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).iotconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_iot_policy" {
			continue
		}

		out, err := conn.ListPolicies(&iot.ListPoliciesInput{})

		if err != nil {
			return err
		}

		for _, t := range out.Policies {
			if *t.PolicyName == rs.Primary.ID {
				return fmt.Errorf("IoT policy still exists:\n%s", t)
			}
		}

	}

	return nil
}

func testAccCheckAWSIoTPolicyExists_basic(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		return nil
	}
}

var testAccAWSIoTPolicy_basic = `
resource "aws_iot_policy" "pubsub" {
  name = "PubSubToAnyTopic"
  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Action": ["iot:*"],
    "Resource": ["*"]
  }]
}
EOF
}

`
