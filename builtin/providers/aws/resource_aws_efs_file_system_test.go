package aws

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/efs"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestResourceAWSEFSReferenceName_validation(t *testing.T) {
	var value string
	var errors []error

	value = acctest.RandString(128)
	_, errors = validateReferenceName(value, "reference_name")
	if len(errors) == 0 {
		t.Fatalf("Expected to trigger a validation error")
	}

	value = acctest.RandString(32)
	_, errors = validateReferenceName(value, "reference_name")
	if len(errors) != 0 {
		t.Fatalf("Expected to trigger a validation error")
	}
}

func TestResourceAWSEFSPerformanceMode_validation(t *testing.T) {
	type testCase struct {
		Value    string
		ErrCount int
	}

	invalidCases := []testCase{
		{
			Value:    "garrusVakarian",
			ErrCount: 1,
		},
		{
			Value:    acctest.RandString(80),
			ErrCount: 1,
		},
	}

	for _, tc := range invalidCases {
		_, errors := validatePerformanceModeType(tc.Value, "performance_mode")
		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected to trigger a validation error")
		}
	}

	validCases := []testCase{
		{
			Value:    "generalPurpose",
			ErrCount: 0,
		},
		{
			Value:    "maxIO",
			ErrCount: 0,
		},
	}

	for _, tc := range validCases {
		_, errors := validatePerformanceModeType(tc.Value, "aws_efs_file_system")
		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected not to trigger a validation error")
		}
	}
}

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
					testAccCheckEfsFileSystemPerformanceMode(
						"aws_efs_file_system.foo",
						"generalPurpose",
					),
				),
			},
			resource.TestStep{
				Config: testAccAWSEFSFileSystemConfigWithTags,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEfsFileSystem(
						"aws_efs_file_system.foo-with-tags",
					),
					testAccCheckEfsFileSystemPerformanceMode(
						"aws_efs_file_system.foo-with-tags",
						"generalPurpose",
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
			resource.TestStep{
				Config: testAccAWSEFSFileSystemConfigWithPerformanceMode,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEfsFileSystem(
						"aws_efs_file_system.foo-with-performance-mode",
					),
					testAccCheckEfsCreationToken(
						"aws_efs_file_system.foo-with-performance-mode",
						"supercalifragilisticexpialidocious",
					),
					testAccCheckEfsFileSystemPerformanceMode(
						"aws_efs_file_system.foo-with-performance-mode",
						"maxIO",
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

		conn := testAccProvider.Meta().(*AWSClient).efsconn
		_, err := conn.DescribeFileSystems(&efs.DescribeFileSystemsInput{
			FileSystemId: aws.String(rs.Primary.ID),
		})

		if err != nil {
			return err
		}

		return nil
	}
}

func testAccCheckEfsCreationToken(resourceID string, expectedToken string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceID]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceID)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).efsconn
		resp, err := conn.DescribeFileSystems(&efs.DescribeFileSystemsInput{
			FileSystemId: aws.String(rs.Primary.ID),
		})

		fs := resp.FileSystems[0]
		if *fs.CreationToken != expectedToken {
			return fmt.Errorf("Creation Token mismatch.\nExpected: %s\nGiven: %v",
				expectedToken, *fs.CreationToken)
		}

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

		conn := testAccProvider.Meta().(*AWSClient).efsconn
		resp, err := conn.DescribeTags(&efs.DescribeTagsInput{
			FileSystemId: aws.String(rs.Primary.ID),
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

func testAccCheckEfsFileSystemPerformanceMode(resourceID string, expectedMode string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceID]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceID)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).efsconn
		resp, err := conn.DescribeFileSystems(&efs.DescribeFileSystemsInput{
			FileSystemId: aws.String(rs.Primary.ID),
		})

		fs := resp.FileSystems[0]
		if *fs.PerformanceMode != expectedMode {
			return fmt.Errorf("Performance Mode mismatch.\nExpected: %s\nGiven: %v",
				expectedMode, *fs.PerformanceMode)
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

const testAccAWSEFSFileSystemConfigWithPerformanceMode = `
resource "aws_efs_file_system" "foo-with-performance-mode" {
	creation_token = "supercalifragilisticexpialidocious"
	performance_mode = "maxIO"
}
`
