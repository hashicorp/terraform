package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/codecommit"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSCodeCommitRepository_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCodeCommitRepositoryDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCodeCommitRepository_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCodeCommitRepositoryExists("aws_codecommit_repository.test"),
				),
			},
		},
	})
}

func TestAccAWSCodeCommitRepository_withChanges(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCodeCommitRepositoryDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCodeCommitRepository_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCodeCommitRepositoryExists("aws_codecommit_repository.test"),
					resource.TestCheckResourceAttr(
						"aws_codecommit_repository.test", "description", "This is a test description"),
				),
			},
			resource.TestStep{
				Config: testAccCodeCommitRepository_withChanges,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCodeCommitRepositoryExists("aws_codecommit_repository.test"),
					resource.TestCheckResourceAttr(
						"aws_codecommit_repository.test", "description", "This is a test description - with changes"),
				),
			},
		},
	})
}

func testAccCheckCodeCommitRepositoryExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		codecommitconn := testAccProvider.Meta().(*AWSClient).codecommitconn
		out, err := codecommitconn.GetRepository(&codecommit.GetRepositoryInput{
			RepositoryName: aws.String(rs.Primary.ID),
		})

		if err != nil {
			return err
		}

		if out.RepositoryMetadata.Arn == nil {
			return fmt.Errorf("No CodeCommit Repository Vault Found")
		}

		if *out.RepositoryMetadata.RepositoryName != rs.Primary.ID {
			return fmt.Errorf("CodeCommit Repository Mismatch - existing: %q, state: %q",
				*out.RepositoryMetadata.RepositoryName, rs.Primary.ID)
		}

		return nil
	}
}

func testAccCheckCodeCommitRepositoryDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).codecommitconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_codecommit_repository" {
			continue
		}

		_, err := conn.GetRepository(&codecommit.GetRepositoryInput{
			RepositoryName: aws.String(rs.Primary.ID),
		})

		if ae, ok := err.(awserr.Error); ok && ae.Code() == "RepositoryDoesNotExistException" {
			continue
		}
		if err == nil {
			return fmt.Errorf("Repository still exists: %s", rs.Primary.ID)
		}
		return err
	}

	return nil
}

const testAccCodeCommitRepository_basic = `
provider "aws" {
  region = "us-east-1"
}
resource "aws_codecommit_repository" "test" {
  repository_name = "my_test_repository"
  description = "This is a test description"
}
`

const testAccCodeCommitRepository_withChanges = `
provider "aws" {
  region = "us-east-1"
}
resource "aws_codecommit_repository" "test" {
  repository_name = "my_test_repository"
  description = "This is a test description - with changes"
}
`
