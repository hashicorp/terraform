package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSS3Bucket(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSS3BucketDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSS3BucketConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSS3BucketExists("aws_s3_bucket.bar"),
				),
			},
		},
	})
}

func testAccCheckAWSS3BucketDestroy(s *terraform.State) error {
	conn := testAccProvider.s3conn

	for _, rs := range s.Resources {
		if rs.Type != "aws_s3_bucket" {
			continue
		}

		bucket := conn.Bucket(rs.ID)
		resp, err := bucket.Head("/")
		if err == nil {
			return fmt.Errorf("S3Bucket still exists")
		}
		defer resp.Body.Close()
	}
	return nil
}

func testAccCheckAWSS3BucketExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.ID == "" {
			return fmt.Errorf("No S3 Bucket ID is set")
		}

		conn := testAccProvider.s3conn
		bucket := conn.Bucket(rs.ID)
		resp, err := bucket.Head("/")
		if err != nil {
			return fmt.Errorf("S3Bucket error: %v", err)
		}
		defer resp.Body.Close()
		return nil
	}
}

const testAccAWSS3BucketConfig = `
resource "aws_s3_bucket" "bar" {
	bucket = "tf-test-bucket"
	acl = "public-read"
}
`
