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

func TestResourceAWSEFSFileSystem_validateReferenceName(t *testing.T) {
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
		t.Fatalf("Expected not to trigger a validation error")
	}
}

func TestResourceAWSEFSFileSystem_validatePerformanceModeType(t *testing.T) {
	_, errors := validatePerformanceModeType("incorrect", "performance_mode")
	if len(errors) == 0 {
		t.Fatalf("Expected to trigger a validation error")
	}

	var testCases = []struct {
		Value    string
		ErrCount int
	}{
		{
			Value:    "generalPurpose",
			ErrCount: 0,
		},
		{
			Value:    "maxIO",
			ErrCount: 0,
		},
	}

	for _, tc := range testCases {
		_, errors := validatePerformanceModeType(tc.Value, "performance_mode")
		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected not to trigger a validation error")
		}
	}
}

func TestResourceAWSEFSFileSystem_hasEmptyFileSystems(t *testing.T) {
	fs := &efs.DescribeFileSystemsOutput{
		FileSystems: []*efs.FileSystemDescription{},
	}

	var actual bool

	actual = hasEmptyFileSystems(fs)
	if !actual {
		t.Fatalf("Expected return value to be true, got %t", actual)
	}

	// Add an empty file system.
	fs.FileSystems = append(fs.FileSystems, &efs.FileSystemDescription{})

	actual = hasEmptyFileSystems(fs)
	if actual {
		t.Fatalf("Expected return value to be false, got %t", actual)
	}

}

func TestAccAWSEFSFileSystem_basic(t *testing.T) {
	rInt := acctest.RandInt()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckEfsFileSystemDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSEFSFileSystemConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"aws_efs_file_system.foo",
						"performance_mode",
						"generalPurpose"),
					testAccCheckEfsFileSystem(
						"aws_efs_file_system.foo",
					),
					testAccCheckEfsFileSystemPerformanceMode(
						"aws_efs_file_system.foo",
						"generalPurpose",
					),
				),
			},
			{
				Config: testAccAWSEFSFileSystemConfigWithTags(rInt),
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
							"Name":    fmt.Sprintf("foo-efs-%d", rInt),
							"Another": "tag",
						},
					),
				),
			},
			{
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

func TestAccAWSEFSFileSystem_pagedTags(t *testing.T) {
	rInt := acctest.RandInt()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckEfsFileSystemDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSEFSFileSystemConfigPagedTags(rInt),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"aws_efs_file_system.foo",
						"tags.%",
						"11"),
					//testAccCheckEfsFileSystem(
					//	"aws_efs_file_system.foo",
					//),
					//testAccCheckEfsFileSystemPerformanceMode(
					//	"aws_efs_file_system.foo",
					//	"generalPurpose",
					//),
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
	creation_token = "radeksimko"
}
`

func testAccAWSEFSFileSystemConfigPagedTags(rInt int) string {
	return fmt.Sprintf(`
	resource "aws_efs_file_system" "foo" {
		tags {
			Name = "foo-efs-%d"
			Another = "tag"
			Test = "yes"
			User = "root"
			Page = "1"
			Environment = "prod"
			CostCenter = "terraform"
			AcceptanceTest = "PagedTags"
			CreationToken = "radek"
			PerfMode = "max"
			Region = "us-west-2"
		}
	}
	`, rInt)
}

func testAccAWSEFSFileSystemConfigWithTags(rInt int) string {
	return fmt.Sprintf(`
	resource "aws_efs_file_system" "foo-with-tags" {
		tags {
			Name = "foo-efs-%d"
			Another = "tag"
		}
	}
	`, rInt)
}

const testAccAWSEFSFileSystemConfigWithPerformanceMode = `
resource "aws_efs_file_system" "foo-with-performance-mode" {
	creation_token = "supercalifragilisticexpialidocious"
	performance_mode = "maxIO"
}
`
