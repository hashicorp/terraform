package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/workspaces"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSWorkspace_basic(t *testing.T) {
	var conf workspaces.Workspace

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSWorkspaceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSWorkspaceConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSWorkspaceExists("aws_workspace.foo", &conf),
				),
			},
		},
	})
}

func testAccCheckAWSWorkspaceExists(n string, res *workspaces.Workspace) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Workspace ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).workspacesconn

		out, err := conn.DescribeWorkspaces(&workspaces.DescribeWorkspacesInput{
			WorkspaceIds: []*string{aws.String(rs.Primary.ID)},
		})
		if err != nil {
			return err
		}

		if len(out.Workspaces) < 1 {
			return fmt.Errorf("No workspace found")
		}

		return nil
	}
}

func testAccCheckAWSWorkspaceDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).workspacesconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_workspace" {
			continue
		}

		out, err := conn.DescribeWorkspaces(&workspaces.DescribeWorkspacesInput{
			WorkspaceIds: []*string{aws.String(rs.Primary.ID)},
		})

		if err != nil {
			// EntityDoesNotExistException means it's gone, this is good
			if wserr, ok := err.(awserr.Error); ok && wserr.Code() == "EntityDoesNotExistException" {
				return nil
			}
			return err
		}

		if out != nil && len(out.Workspaces) > 0 {
			return fmt.Errorf("Expected AWS Workspace to be gone, but was still found")
		}

		return nil
	}

	return fmt.Errorf("Default error in Workspace Test")
}

const testAccAWSWorkspaceConfig = `
resource "aws_directory_service_directory" "bar" {
  name = "corp.notexample.com"
  password = "SuperSecretPassw0rd"
  size = "Small"

  vpc_settings {
    vpc_id = "${aws_vpc.main.id}"
    subnet_ids = ["${aws_subnet.foo.id}", "${aws_subnet.bar.id}"]
  }
}

resource "aws_vpc" "main" {
  cidr_block = "10.0.0.0/16"
}

resource "aws_subnet" "foo" {
  vpc_id = "${aws_vpc.main.id}"
  availability_zone = "eu-west-1a"
  cidr_block = "10.0.1.0/24"
}
resource "aws_subnet" "bar" {
  vpc_id = "${aws_vpc.main.id}"
  availability_zone = "eu-west-1b"
  cidr_block = "10.0.2.0/24"
}

resource "aws_workspace" "foo" {
  bundle_name  = "Value"
  directory_id = "${aws_directory_service_directory.bar.id}"
  user_name    = "Administrator"
}
`
