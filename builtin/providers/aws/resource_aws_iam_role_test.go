package aws

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSRole_basic(t *testing.T) {
	var conf iam.GetRoleOutput

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSRoleDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSRoleConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSRoleExists("aws_iam_role.role", &conf),
					testAccCheckAWSRoleAttributes(&conf),
				),
			},
		},
	})
}

func TestAccAWSRole_testNameChange(t *testing.T) {
	var conf iam.GetRoleOutput

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSRoleDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSRolePre,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSRoleExists("aws_iam_role.role_update_test", &conf),
				),
			},

			resource.TestStep{
				Config: testAccAWSRolePost,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSRoleExists("aws_iam_role.role_update_test", &conf),
				),
			},
		},
	})
}

func TestAccAWSRole_testAssumeRolePolicyChange(t *testing.T) {
	var conf iam.GetRoleOutput

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSRoleDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSRoleConfigPolicyChangePre,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSRoleExists("aws_iam_role.role", &conf),
					testAccCheckAWSRoleAssumeRolePolicy(
						&conf,
						`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"Service":"ec2.amazonaws.com"},"Action":"sts:AssumeRole"}]}`,
					),
				),
			},

			resource.TestStep{
				Config: testAccAWSRoleConfigPolicyChangePost,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSRoleExists("aws_iam_role.role", &conf),
					testAccCheckAWSRoleAssumeRolePolicy(
						&conf,
						`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"Service":"s3.amazonaws.com"},"Action":"sts:AssumeRole"}]}`,
					),
				),
			},
		},
	})
}

func testAccCheckAWSRoleDestroy(s *terraform.State) error {
	iamconn := testAccProvider.Meta().(*AWSClient).iamconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_iam_role" {
			continue
		}

		// Try to get role
		_, err := iamconn.GetRole(&iam.GetRoleInput{
			RoleName: aws.String(rs.Primary.ID),
		})
		if err == nil {
			return fmt.Errorf("still exist.")
		}

		// Verify the error is what we want
		ec2err, ok := err.(awserr.Error)
		if !ok {
			return err
		}
		if ec2err.Code() != "NoSuchEntity" {
			return err
		}
	}

	return nil
}

func testAccCheckAWSRoleExists(n string, res *iam.GetRoleOutput) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Role name is set")
		}

		iamconn := testAccProvider.Meta().(*AWSClient).iamconn

		resp, err := iamconn.GetRole(&iam.GetRoleInput{
			RoleName: aws.String(rs.Primary.ID),
		})
		if err != nil {
			return err
		}

		*res = *resp

		return nil
	}
}

func testAccCheckAWSRoleAssumeRolePolicy(role *iam.GetRoleOutput, expectedPolicy string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		policy, err := url.QueryUnescape(*role.Role.AssumeRolePolicyDocument)
		if err != nil {
			return fmt.Errorf("could not decode policy: %s", *role.Role.AssumeRolePolicyDocument)
		}
		if policy != expectedPolicy {
			return fmt.Errorf("expected assume role policy to be %s, got %s", expectedPolicy, policy)
		}
		return nil
	}
}

func testAccCheckAWSRoleAttributes(role *iam.GetRoleOutput) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if *role.Role.RoleName != "test-role" {
			return fmt.Errorf("Bad name: %s", *role.Role.RoleName)
		}

		if *role.Role.Path != "/" {
			return fmt.Errorf("Bad path: %s", *role.Role.Path)
		}
		return nil
	}
}

const testAccAWSRoleConfig = `
resource "aws_iam_role" "role" {
	name   = "test-role"
	path = "/"
	assume_role_policy = "{\"Version\":\"2012-10-17\",\"Statement\":[{\"Effect\":\"Allow\",\"Principal\":{\"Service\":[\"ec2.amazonaws.com\"]},\"Action\":[\"sts:AssumeRole\"]}]}"
}
`

const testAccAWSRoleConfigPolicyChangePre = `
resource "aws_iam_role" "role" {
	name   = "test-role"
	path = "/"
	assume_role_policy = "{\"Version\":\"2012-10-17\",\"Statement\":[{\"Effect\":\"Allow\",\"Principal\":{\"Service\":\"ec2.amazonaws.com\"},\"Action\":\"sts:AssumeRole\"}]}"
}
`

const testAccAWSRoleConfigPolicyChangePost = `
resource "aws_iam_role" "role" {
	name   = "test-role"
	path = "/"
	assume_role_policy = "{\"Version\":\"2012-10-17\",\"Statement\":[{\"Effect\":\"Allow\",\"Principal\":{\"Service\":\"s3.amazonaws.com\"},\"Action\":\"sts:AssumeRole\"}]}"
}
`

const testAccAWSRolePre = `
resource "aws_iam_role" "role_update_test" {
  name = "tf_old_name"
  path = "/test/"
  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "ec2.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF
}

resource "aws_iam_role_policy" "role_update_test" {
  name = "role_update_test"
  role = "${aws_iam_role.role_update_test.id}"
  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "s3:GetBucketLocation",
        "s3:ListAllMyBuckets"
      ],
      "Resource": "arn:aws:s3:::*"
    }
  ]
}
EOF
}

resource "aws_iam_instance_profile" "role_update_test" {
  name = "role_update_test"
  path = "/test/"
  roles = ["${aws_iam_role.role_update_test.name}"]
}

`

const testAccAWSRolePost = `
resource "aws_iam_role" "role_update_test" {
  name = "tf_new_name"
  path = "/test/"
  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "ec2.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF
}

resource "aws_iam_role_policy" "role_update_test" {
  name = "role_update_test"
  role = "${aws_iam_role.role_update_test.id}"
  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "s3:GetBucketLocation",
        "s3:ListAllMyBuckets"
      ],
      "Resource": "arn:aws:s3:::*"
    }
  ]
}
EOF
}

resource "aws_iam_instance_profile" "role_update_test" {
  name = "role_update_test"
  path = "/test/"
  roles = ["${aws_iam_role.role_update_test.name}"]
}

`
