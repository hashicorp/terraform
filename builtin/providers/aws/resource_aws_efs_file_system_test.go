package aws

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/efs"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSEFSFileSystem(t *testing.T) {
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
	if len(s.RootModule().Resources) > 0 {
		return fmt.Errorf("Expected all resources to be gone, but found: %#v", s.RootModule().Resources)
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
