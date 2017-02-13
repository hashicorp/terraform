package aws

import (
	"fmt"
	"strings"
	"testing"

	"regexp"

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
			{
				Config: testAccAWSRoleConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSRoleExists("aws_iam_role.role", &conf),
					testAccCheckAWSRoleAttributes(&conf),
					resource.TestCheckResourceAttrSet(
						"aws_iam_role.role", "create_date",
					),
				),
			},
		},
	})
}

func TestAccAWSRole_namePrefix(t *testing.T) {
	var conf iam.GetRoleOutput

	resource.Test(t, resource.TestCase{
		PreCheck:        func() { testAccPreCheck(t) },
		IDRefreshName:   "aws_iam_role.role",
		IDRefreshIgnore: []string{"name_prefix"},
		Providers:       testAccProviders,
		CheckDestroy:    testAccCheckAWSRoleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSRolePrefixNameConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSRoleExists("aws_iam_role.role", &conf),
					testAccCheckAWSRoleGeneratedNamePrefix(
						"aws_iam_role.role", "test-role-"),
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
			{
				Config: testAccAWSRolePre,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSRoleExists("aws_iam_role.role_update_test", &conf),
				),
			},

			{
				Config: testAccAWSRolePost,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSRoleExists("aws_iam_role.role_update_test", &conf),
				),
			},
		},
	})
}

func TestAccAWSRole_badJSON(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSRoleDestroy,
		Steps: []resource.TestStep{
			{
				Config:      testAccAWSRoleConfig_badJson,
				ExpectError: regexp.MustCompile(`"assume_role_policy" contains an invalid JSON:.*`),
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

func testAccCheckAWSRoleGeneratedNamePrefix(resource, prefix string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		r, ok := s.RootModule().Resources[resource]
		if !ok {
			return fmt.Errorf("Resource not found")
		}
		name, ok := r.Primary.Attributes["name"]
		if !ok {
			return fmt.Errorf("Name attr not found: %#v", r.Primary.Attributes)
		}
		if !strings.HasPrefix(name, prefix) {
			return fmt.Errorf("Name: %q, does not have prefix: %q", name, prefix)
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

const testAccAWSRolePrefixNameConfig = `
resource "aws_iam_role" "role" {
  name_prefix = "test-role-"
  path = "/"
  assume_role_policy = "{\"Version\":\"2012-10-17\",\"Statement\":[{\"Effect\":\"Allow\",\"Principal\":{\"Service\":[\"ec2.amazonaws.com\"]},\"Action\":[\"sts:AssumeRole\"]}]}"
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

const testAccAWSRoleConfig_badJson = `
	resource "aws_iam_role" "my_instance_role" {
  name = "test-role"

  assume_role_policy = <<POLICY
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Action": "sts:AssumeRole",
            "Principal": {
                "Service": "ec2.amazonaws.com",
            },
            "Effect": "Allow",
            "Sid": ""
        }
    ]
}
POLICY
}`
