package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSIAMPolicy_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckIAMPolicyDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccIAMPolicyConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIAMPolicyExists(
						"aws_iam_policy.foo",
					),
				),
			},
			resource.TestStep{
				Config: testAccIAMPolicyConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIAMPolicyExists(
						"aws_iam_policy.foo",
					),
				),
			},
		},
	})
}

func testAccCheckIAMPolicyExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		arn := rs.Primary.ID
		//policy = policy.Attributes["policy"]

		conn := testAccProvider.Meta().(*AWSClient).iamconn

		request := &iam.GetPolicyInput{
			PolicyArn: aws.String(arn),
		}

		_, err := conn.GetPolicy(request)

		if err != nil {
			return err
		}

		return nil
	}
}

func testAccCheckIAMPolicyDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).iamconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_iam_policy" {
			continue
		}

		arn := rs.Primary.ID

		request := &iam.GetPolicyInput{
			PolicyArn: aws.String(arn),
		}

		_, err := conn.GetPolicy(request)
		if err != nil {
			// Verify the error is what we want
			if ae, ok := err.(awserr.Error); ok && ae.Code() == "NoSuchEntity" {
				continue
			}
			return err
		}

		return fmt.Errorf("IAM policy still exists")
	}

	return nil
}

const testAccIAMPolicyConfig = `
resource "aws_iam_policy" "foo" {
	name = "foo_policy"
	policy = "{\"Version\":\"2012-10-17\",\"Statement\":{\"Effect\":\"Allow\",\"Action\":\"*\",\"Resource\":\"*\"}}"
}
`

const testAccIAMPolicyConfigUpdate = `
resource "aws_iam_policy" "foo" {
	name = "foo_policy"
	policy = "{\"Version\":\"2012-10-17\",\"Statement\":{\"Effect\":\"Allow\",\"Action\":\"ec2:Describe*\",\"Resource\":\"*\"}}"
}
`
