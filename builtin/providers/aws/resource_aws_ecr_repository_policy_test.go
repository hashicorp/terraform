package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSEcrRepositoryPolicy_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSEcrRepositoryPolicyDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSEcrRepositoryPolicy,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcrRepositoryPolicyExists("aws_ecr_repository_policy.default"),
				),
			},
		},
	})
}

func testAccCheckAWSEcrRepositoryPolicyDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).ecrconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_ecr_repository_policy" {
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

func testAccCheckAWSEcrRepositoryPolicyExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		return nil
	}
}

var testAccAWSEcrRepositoryPolicy = `
# ECR initially only available in us-east-1
# https://aws.amazon.com/blogs/aws/ec2-container-registry-now-generally-available/
provider "aws" {
	region = "us-east-1"
}
resource "aws_ecr_repository" "foo" {
	name = "bar"
}

resource "aws_ecr_repository_policy" "default" {
	repository = "${aws_ecr_repository.foo.name}"
	policy = <<EOF
{
    "Version": "2008-10-17",
    "Statement": [
        {
            "Sid": "testpolicy",
            "Effect": "Allow",
            "Principal": "*",
            "Action": [
                "ecr:ListImages"
            ]
        }
    ]
}
EOF
}
`
