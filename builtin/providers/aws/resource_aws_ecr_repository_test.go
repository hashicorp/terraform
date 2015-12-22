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

func TestAccAWSEcrRepository_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSEcrRepositoryDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSEcrRepository,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcrRepositoryExists("aws_ecr_repository.default"),
				),
			},
		},
	})
}

func testAccCheckAWSEcrRepositoryDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).ecrconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_ecr_repository" {
			continue
		}

		input := ecr.DescribeRepositoriesInput{
			RegistryId:      aws.String(rs.Primary.Attributes["registry_id"]),
			RepositoryNames: []*string{aws.String(rs.Primary.Attributes["name"])},
		}

		out, err := conn.DescribeRepositories(&input)

		if err != nil {
			if ecrerr, ok := err.(awserr.Error); ok && ecrerr.Code() == "RepositoryNotFoundException" {
				return nil
			}
			return err
		}

		for _, repository := range out.Repositories {
			if repository.RepositoryName == aws.String(rs.Primary.Attributes["name"]) {
				return fmt.Errorf("ECR repository still exists:\n%#v", repository)
			}
		}
	}

	return nil
}

func testAccCheckAWSEcrRepositoryExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		return nil
	}
}

var testAccAWSEcrRepository = `
# ECR initially only available in us-east-1
# https://aws.amazon.com/blogs/aws/ec2-container-registry-now-generally-available/
provider "aws" {
	region = "us-east-1"
}
resource "aws_ecr_repository" "default" {
	name = "foo-repository-terraform"
}
`
