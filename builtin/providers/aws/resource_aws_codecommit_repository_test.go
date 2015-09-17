package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
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
	if len(s.RootModule().Resources) > 0 {
		return fmt.Errorf("Expected all resources to be gone, but found: %#v",
			s.RootModule().Resources)
	}

	return nil
}

const testAccCodeCommitRepository_basic = `
resource "aws_codecommit_repository" "test" {
  repository_name = "my_test_repository"
  description = "This is a test description"
}
`
