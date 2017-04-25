package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSIAMGroupPolicy_basic(t *testing.T) {
	rInt := acctest.RandInt()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckIAMGroupPolicyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccIAMGroupPolicyConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIAMGroupPolicy(
						"aws_iam_group.group",
						"aws_iam_group_policy.foo",
					),
				),
			},
			{
				Config: testAccIAMGroupPolicyConfigUpdate(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIAMGroupPolicy(
						"aws_iam_group.group",
						"aws_iam_group_policy.bar",
					),
				),
			},
		},
	})
}

func TestAccAWSIAMGroupPolicy_namePrefix(t *testing.T) {
	rInt := acctest.RandInt()
	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_iam_group_policy.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckIAMGroupPolicyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccIAMGroupPolicyConfig_namePrefix(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIAMGroupPolicy(
						"aws_iam_group.test",
						"aws_iam_group_policy.test",
					),
				),
			},
		},
	})
}

func TestAccAWSIAMGroupPolicy_generatedName(t *testing.T) {
	rInt := acctest.RandInt()
	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_iam_group_policy.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckIAMGroupPolicyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccIAMGroupPolicyConfig_generatedName(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIAMGroupPolicy(
						"aws_iam_group.test",
						"aws_iam_group_policy.test",
					),
				),
			},
		},
	})
}

func testAccCheckIAMGroupPolicyDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).iamconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_iam_group_policy" {
			continue
		}

		group, name := resourceAwsIamGroupPolicyParseId(rs.Primary.ID)

		request := &iam.GetGroupPolicyInput{
			PolicyName: aws.String(name),
			GroupName:  aws.String(group),
		}

		_, err := conn.GetGroupPolicy(request)
		if err != nil {
			// Verify the error is what we want
			if ae, ok := err.(awserr.Error); ok && ae.Code() == "NoSuchEntity" {
				continue
			}
			return err
		}

		return fmt.Errorf("still exists")
	}

	return nil
}

func testAccCheckIAMGroupPolicy(
	iamGroupResource string,
	iamGroupPolicyResource string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[iamGroupResource]
		if !ok {
			return fmt.Errorf("Not Found: %s", iamGroupResource)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		policy, ok := s.RootModule().Resources[iamGroupPolicyResource]
		if !ok {
			return fmt.Errorf("Not Found: %s", iamGroupPolicyResource)
		}

		iamconn := testAccProvider.Meta().(*AWSClient).iamconn
		group, name := resourceAwsIamGroupPolicyParseId(policy.Primary.ID)
		_, err := iamconn.GetGroupPolicy(&iam.GetGroupPolicyInput{
			GroupName:  aws.String(group),
			PolicyName: aws.String(name),
		})

		if err != nil {
			return err
		}

		return nil
	}
}

func testAccIAMGroupPolicyConfig(rInt int) string {
	return fmt.Sprintf(`
	resource "aws_iam_group" "group" {
		name = "test_group_%d"
		path = "/"
	}

	resource "aws_iam_group_policy" "foo" {
		name = "foo_policy_%d"
		group = "${aws_iam_group.group.name}"
		policy = <<EOF
{
	"Version": "2012-10-17",
	"Statement": {
		"Effect": "Allow",
		"Action": "*",
		"Resource": "*"
	}
}
EOF
	}`, rInt, rInt)
}

func testAccIAMGroupPolicyConfig_namePrefix(rInt int) string {
	return fmt.Sprintf(`
	resource "aws_iam_group" "test" {
		name = "test_group_%d"
		path = "/"
	}

	resource "aws_iam_group_policy" "test" {
		name_prefix = "test-%d"
		group = "${aws_iam_group.test.name}"
		policy = <<EOF
{
	"Version": "2012-10-17",
	"Statement": {
		"Effect": "Allow",
		"Action": "*",
		"Resource": "*"
	}
}
EOF
	}`, rInt, rInt)
}

func testAccIAMGroupPolicyConfig_generatedName(rInt int) string {
	return fmt.Sprintf(`
	resource "aws_iam_group" "test" {
		name = "test_group_%d"
		path = "/"
	}

	resource "aws_iam_group_policy" "test" {
		group = "${aws_iam_group.test.name}"
		policy = <<EOF
{
	"Version": "2012-10-17",
	"Statement": {
		"Effect": "Allow",
		"Action": "*",
		"Resource": "*"
	}
}
EOF
	}`, rInt)
}

func testAccIAMGroupPolicyConfigUpdate(rInt int) string {
	return fmt.Sprintf(`
	resource "aws_iam_group" "group" {
		name = "test_group_%d"
		path = "/"
	}

	resource "aws_iam_group_policy" "foo" {
		name = "foo_policy_%d"
		group = "${aws_iam_group.group.name}"
		policy = "{\"Version\":\"2012-10-17\",\"Statement\":{\"Effect\":\"Allow\",\"Action\":\"*\",\"Resource\":\"*\"}}"
	}

	resource "aws_iam_group_policy" "bar" {
		name = "bar_policy_%d"
		group = "${aws_iam_group.group.name}"
		policy = "{\"Version\":\"2012-10-17\",\"Statement\":{\"Effect\":\"Allow\",\"Action\":\"*\",\"Resource\":\"*\"}}"
	}`, rInt, rInt, rInt)
}
