package aws

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/efs"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSEFSFileSystem_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckEfsFileSystemDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSEFSFileSystemConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEfsFileSystem(
						"aws_efs_file_system.foo",
					),
				),
			},
			resource.TestStep{
				Config: testAccAWSEFSFileSystemConfigWithTags,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEfsFileSystem(
						"aws_efs_file_system.foo-with-tags",
					),
					testAccCheckEfsFileSystemTags(
						"aws_efs_file_system.foo-with-tags",
						map[string]string{
							"Name":    "foo-efs",
							"Another": "tag",
						},
					),
				),
			},
		},
	})
}

func testAccCheckEfsFileSystemDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).efsconn
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_efs_file_system" {
			continue
		}

		resp, err := conn.DescribeFileSystems(&efs.DescribeFileSystemsInput{
			FileSystemId: aws.String(rs.Primary.ID),
		})
		if err != nil {
			if efsErr, ok := err.(awserr.Error); ok && efsErr.Code() == "FileSystemNotFound" {
				// gone
				return nil
			}
			return fmt.Errorf("Error describing EFS in tests: %s", err)
		}
		if len(resp.FileSystems) > 0 {
			return fmt.Errorf("EFS file system %q still exists", rs.Primary.ID)
		}
	}

	return nil
}

func testAccCheckEfsFileSystem(resourceID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceID]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceID)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		fs, ok := s.RootModule().Resources[resourceID]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceID)
		}

		conn := testAccProvider.Meta().(*AWSClient).efsconn
		_, err := conn.DescribeFileSystems(&efs.DescribeFileSystemsInput{
			FileSystemId: aws.String(fs.Primary.ID),
		})

		if err != nil {
			return err
		}

		return nil
	}
}

func testAccCheckEfsFileSystemTags(resourceID string, expectedTags map[string]string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceID]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceID)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		fs, ok := s.RootModule().Resources[resourceID]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceID)
		}

		conn := testAccProvider.Meta().(*AWSClient).efsconn
		resp, err := conn.DescribeTags(&efs.DescribeTagsInput{
			FileSystemId: aws.String(fs.Primary.ID),
		})

		if !reflect.DeepEqual(expectedTags, tagsToMapEFS(resp.Tags)) {
			return fmt.Errorf("Tags mismatch.\nExpected: %#v\nGiven: %#v",
				expectedTags, resp.Tags)
		}

		if err != nil {
			return err
		}

		return nil
	}
}

const testAccAWSEFSFileSystemConfig = `
resource "aws_efs_file_system" "foo" {
	reference_name = "radeksimko"
}
`

const testAccAWSEFSFileSystemConfigWithTags = `
resource "aws_efs_file_system" "foo-with-tags" {
	reference_name = "yada_yada"
	tags {
		Name = "foo-efs"
		Another = "tag"
	}
}
`
