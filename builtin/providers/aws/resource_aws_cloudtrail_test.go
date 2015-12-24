package aws

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudtrail"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSCloudTrail_basic(t *testing.T) {
	var trail cloudtrail.Trail

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCloudTrailDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSCloudTrailConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudTrailExists("aws_cloudtrail.foobar", &trail),
					resource.TestCheckResourceAttr("aws_cloudtrail.foobar", "include_global_service_events", "true"),
				),
			},
			resource.TestStep{
				Config: testAccAWSCloudTrailConfigModified,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudTrailExists("aws_cloudtrail.foobar", &trail),
					resource.TestCheckResourceAttr("aws_cloudtrail.foobar", "s3_key_prefix", "/prefix"),
					resource.TestCheckResourceAttr("aws_cloudtrail.foobar", "include_global_service_events", "false"),
				),
			},
		},
	})
}

func TestAccAWSCloudTrail_enable_logging(t *testing.T) {
	var trail cloudtrail.Trail

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCloudTrailDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSCloudTrailConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudTrailExists("aws_cloudtrail.foobar", &trail),
					// AWS will create the trail with logging turned off.
					// Test that "enable_logging" default works.
					testAccCheckCloudTrailLoggingEnabled("aws_cloudtrail.foobar", true, &trail),
				),
			},
			resource.TestStep{
				Config: testAccAWSCloudTrailConfigModified,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudTrailExists("aws_cloudtrail.foobar", &trail),
					testAccCheckCloudTrailLoggingEnabled("aws_cloudtrail.foobar", false, &trail),
				),
			},
			resource.TestStep{
				Config: testAccAWSCloudTrailConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudTrailExists("aws_cloudtrail.foobar", &trail),
					testAccCheckCloudTrailLoggingEnabled("aws_cloudtrail.foobar", true, &trail),
				),
			},
		},
	})
}

func testAccCheckCloudTrailExists(n string, trail *cloudtrail.Trail) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn := testAccProvider.Meta().(*AWSClient).cloudtrailconn
		params := cloudtrail.DescribeTrailsInput{
			TrailNameList: []*string{aws.String(rs.Primary.ID)},
		}
		resp, err := conn.DescribeTrails(&params)
		if err != nil {
			return err
		}
		if len(resp.TrailList) == 0 {
			return fmt.Errorf("Trail not found")
		}
		*trail = *resp.TrailList[0]

		return nil
	}
}

func testAccCheckCloudTrailLoggingEnabled(n string, desired bool, trail *cloudtrail.Trail) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn := testAccProvider.Meta().(*AWSClient).cloudtrailconn
		params := cloudtrail.GetTrailStatusInput{
			Name: aws.String(rs.Primary.ID),
		}
		resp, err := conn.GetTrailStatus(&params)

		if err != nil {
			return err
		}
		if *resp.IsLogging != desired {
			return fmt.Errorf("Expected logging status %t, given %t", desired, *resp.IsLogging)
		}

		return nil
	}
}

func testAccCheckAWSCloudTrailDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).cloudtrailconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_cloudtrail" {
			continue
		}

		params := cloudtrail.DescribeTrailsInput{
			TrailNameList: []*string{aws.String(rs.Primary.ID)},
		}

		resp, err := conn.DescribeTrails(&params)

		if err == nil {
			if len(resp.TrailList) != 0 &&
				*resp.TrailList[0].Name == rs.Primary.ID {
				return fmt.Errorf("CloudTrail still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

var cloudTrailRandInt = rand.New(rand.NewSource(time.Now().UnixNano())).Int()

var testAccAWSCloudTrailConfig = fmt.Sprintf(`
resource "aws_cloudtrail" "foobar" {
    name = "tf-trail-foobar"
    s3_bucket_name = "${aws_s3_bucket.foo.id}"
}

resource "aws_s3_bucket" "foo" {
	bucket = "tf-test-trail-%d"
	force_destroy = true
	policy = <<POLICY
{
	"Version": "2012-10-17",
	"Statement": [
		{
			"Sid": "AWSCloudTrailAclCheck",
			"Effect": "Allow",
			"Principal": "*",
			"Action": "s3:GetBucketAcl",
			"Resource": "arn:aws:s3:::tf-test-trail-%d"
		},
		{
			"Sid": "AWSCloudTrailWrite",
			"Effect": "Allow",
			"Principal": "*",
			"Action": "s3:PutObject",
			"Resource": "arn:aws:s3:::tf-test-trail-%d/*",
			"Condition": {
				"StringEquals": {
					"s3:x-amz-acl": "bucket-owner-full-control"
				}
			}
		}
	]
}
POLICY
}
`, cloudTrailRandInt, cloudTrailRandInt, cloudTrailRandInt)

var testAccAWSCloudTrailConfigModified = fmt.Sprintf(`
resource "aws_cloudtrail" "foobar" {
    name = "tf-trail-foobar"
    s3_bucket_name = "${aws_s3_bucket.foo.id}"
    s3_key_prefix = "/prefix"
    include_global_service_events = false
    enable_logging = false
}

resource "aws_s3_bucket" "foo" {
	bucket = "tf-test-trail-%d"
	force_destroy = true
	policy = <<POLICY
{
	"Version": "2012-10-17",
	"Statement": [
		{
			"Sid": "AWSCloudTrailAclCheck",
			"Effect": "Allow",
			"Principal": "*",
			"Action": "s3:GetBucketAcl",
			"Resource": "arn:aws:s3:::tf-test-trail-%d"
		},
		{
			"Sid": "AWSCloudTrailWrite",
			"Effect": "Allow",
			"Principal": "*",
			"Action": "s3:PutObject",
			"Resource": "arn:aws:s3:::tf-test-trail-%d/*",
			"Condition": {
				"StringEquals": {
					"s3:x-amz-acl": "bucket-owner-full-control"
				}
			}
		}
	]
}
POLICY
}
`, cloudTrailRandInt, cloudTrailRandInt, cloudTrailRandInt)
