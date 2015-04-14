package aws

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/service/s3"
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
	conn := testAccProvider.Meta().(*AWSClient).s3conn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_s3_bucket" {
			continue
		}
		_, err := conn.DeleteBucket(&s3.DeleteBucketInput{
			Bucket: aws.String(rs.Primary.ID),
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func testAccCheckAWSS3BucketExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No S3 Bucket ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).s3conn
		_, err := conn.HeadBucket(&s3.HeadBucketInput{
			Bucket: aws.String(rs.Primary.ID),
		})

		if err != nil {
			return fmt.Errorf("S3Bucket error: %v", err)
		}
		return nil
	}
}

// This needs a bit of randoness as the name can only be
// used once globally within AWS
var testAccAWSS3BucketConfig = fmt.Sprintf(`
resource "aws_s3_bucket" "bar" {
	bucket = "tf-test-bucket-%d"
	acl = "public-read"
}
`, rand.New(rand.NewSource(time.Now().UnixNano())).Int())
