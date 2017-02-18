package aws

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSEcrRepositoryPolicyStatement_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSEcrRepositoryPolicyStatementDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSEcrRepositoryPolicyStatement,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcrRepositoryPolicyStatementExists("aws_ecr_repository_policy_statement.default"),
					testAccCheckAWSEcrRepositoryPolicyStatementExists("aws_ecr_repository_policy_statement.other"),
				),
			},
			resource.TestStep{
				Config: testAccAWSEcrRepositoryPolicyStatementWithoutOther,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcrRepositoryPolicyStatementIntegrity("aws_ecr_repository.foo", "testpolicy", "anotherpolicy"),
				),
			},
		},
	})
}

func testAccCheckAWSEcrRepositoryPolicyStatementDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).ecrconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_ecr_repository_policy_statement" {
			continue
		}

		_, err := conn.GetRepositoryPolicy(&ecr.GetRepositoryPolicyInput{
			RegistryId:     aws.String(rs.Primary.Attributes["registry_id"]),
			RepositoryName: aws.String(rs.Primary.Attributes["repository"]),
		})

		if err != nil {
			if ecrerr, ok := err.(awserr.Error); ok && ecrerr.Code() == "RepositoryNotFoundException" {
				return nil
			}
			return err
		}
	}

	return nil
}

func testAccCheckAWSEcrRepositoryPolicyStatementExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*AWSClient).ecrconn
		rs, ok := s.RootModule().Resources[name]

		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		out, err := conn.GetRepositoryPolicy(&ecr.GetRepositoryPolicyInput{
			RegistryId:     aws.String(rs.Primary.Attributes["registry_id"]),
			RepositoryName: aws.String(rs.Primary.Attributes["repository"]),
		})

		if err != nil {
			return err
		}

		policy := make(map[string]interface{})
		if err := json.Unmarshal([]byte(*out.PolicyText), &policy); err != nil {
			return err
		}

		stmts := policy["Statement"].([]interface{})
		found := false

		for _, s := range stmts {
			stmt := s.(map[string]interface{})

			if stmt["Sid"] == rs.Primary.Attributes["sid"] {
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("Not found on AWS: %s", name)
		}

		return nil
	}
}

func testAccCheckAWSEcrRepositoryPolicyStatementIntegrity(repo, sid, removedSid string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*AWSClient).ecrconn
		rs, ok := s.RootModule().Resources[repo]

		if !ok {
			return fmt.Errorf("Repository not found: %s", repo)
		}

		out, err := conn.GetRepositoryPolicy(&ecr.GetRepositoryPolicyInput{
			RegistryId:     aws.String(rs.Primary.Attributes["registry_id"]),
			RepositoryName: aws.String(rs.Primary.Attributes["name"]),
		})

		if err != nil {
			return err
		}

		policy := make(map[string]interface{})
		if err := json.Unmarshal([]byte(*out.PolicyText), &policy); err != nil {
			return err
		}

		stmts := policy["Statement"].([]interface{})
		sidFound := false
		removedFound := false

		for _, s := range stmts {
			stmt := s.(map[string]interface{})

			if stmt["Sid"] == sid {
				sidFound = true
			}

			if stmt["Sid"] == removedSid {
				removedFound = true
			}
		}

		if !sidFound {
			return fmt.Errorf("Not found on AWS: %s", sid)
		}

		if removedFound {
			return fmt.Errorf("Dangling resource found on AWS: %s", removedSid)
		}

		return nil
	}
}

var testAccAWSEcrRepositoryPolicyStatement = `
# ECR initially only available in us-east-1
# https://aws.amazon.com/blogs/aws/ec2-container-registry-now-generally-available/
provider "aws" {
	region = "us-east-1"
}

resource "aws_ecr_repository" "foo" {
	name = "bar"
}

resource "aws_ecr_repository_policy_statement" "default" {
	repository = "${aws_ecr_repository.foo.name}"
	sid = "testpolicy"
	statement = <<EOF
{
	"Effect": "Allow",
	"Principal": "*",
	"Action": [
		"ecr:ListImages"
	]
}
EOF
}

resource "aws_ecr_repository_policy_statement" "other" {
	repository = "${aws_ecr_repository.foo.name}"
	sid = "anotherpolicy"
	statement = <<EOF
{
	"Effect": "Deny",
	"Principal": "*",
	"Action": [
		"ecr:ListImages"
	]
}
EOF
}
`

var testAccAWSEcrRepositoryPolicyStatementWithoutOther = `
# ECR initially only available in us-east-1
# https://aws.amazon.com/blogs/aws/ec2-container-registry-now-generally-available/
provider "aws" {
	region = "us-east-1"
}

resource "aws_ecr_repository" "foo" {
	name = "bar"
}

resource "aws_ecr_repository_policy_statement" "default" {
	repository = "${aws_ecr_repository.foo.name}"
	sid = "testpolicy"
	statement = <<EOF
{
	"Effect": "Allow",
	"Principal": "*",
	"Action": [
		"ecr:ListImages"
	]
}
EOF
}
`
