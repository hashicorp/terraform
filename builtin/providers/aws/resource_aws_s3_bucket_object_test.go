package aws

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

var tf, err = ioutil.TempFile("", "tf")

func TestAccAWSS3BucketObject_source(t *testing.T) {
	// first write some data to the tempfile just so it's not 0 bytes.
	ioutil.WriteFile(tf.Name(), []byte("{anything will do }"), 0644)
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			if err != nil {
				panic(err)
			}
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSS3BucketObjectDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSS3BucketObjectConfigSource,
				Check:  testAccCheckAWSS3BucketObjectExists("aws_s3_bucket_object.object"),
			},
		},
	})
}

func TestAccAWSS3BucketObject_content(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			if err != nil {
				panic(err)
			}
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSS3BucketObjectDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSS3BucketObjectConfigContent,
				Check:  testAccCheckAWSS3BucketObjectExists("aws_s3_bucket_object.object"),
			},
		},
	})
}

func TestAccAWSS3BucketObject_withContentCharacteristics(t *testing.T) {
	// first write some data to the tempfile just so it's not 0 bytes.
	ioutil.WriteFile(tf.Name(), []byte("{anything will do }"), 0644)
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			if err != nil {
				panic(err)
			}
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSS3BucketObjectDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSS3BucketObjectConfig_withContentCharacteristics,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSS3BucketObjectExists("aws_s3_bucket_object.object"),
					resource.TestCheckResourceAttr(
						"aws_s3_bucket_object.object", "content_type", "binary/octet-stream"),
				),
			},
		},
	})
}

func testAccCheckAWSS3BucketObjectDestroy(s *terraform.State) error {
	s3conn := testAccProvider.Meta().(*AWSClient).s3conn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_s3_bucket_object" {
			continue
		}

		_, err := s3conn.HeadObject(
			&s3.HeadObjectInput{
				Bucket:  aws.String(rs.Primary.Attributes["bucket"]),
				Key:     aws.String(rs.Primary.Attributes["key"]),
				IfMatch: aws.String(rs.Primary.Attributes["etag"]),
			})
		if err == nil {
			return fmt.Errorf("AWS S3 Object still exists: %s", rs.Primary.ID)
		}
	}
	return nil
}

func testAccCheckAWSS3BucketObjectExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		defer os.Remove(tf.Name())

		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not Found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No S3 Bucket Object ID is set")
		}

		s3conn := testAccProvider.Meta().(*AWSClient).s3conn
		_, err := s3conn.GetObject(
			&s3.GetObjectInput{
				Bucket:  aws.String(rs.Primary.Attributes["bucket"]),
				Key:     aws.String(rs.Primary.Attributes["key"]),
				IfMatch: aws.String(rs.Primary.Attributes["etag"]),
			})
		if err != nil {
			return fmt.Errorf("S3Bucket Object error: %s", err)
		}
		return nil
	}
}

var randomBucket = randInt
var testAccAWSS3BucketObjectConfigSource = fmt.Sprintf(`
resource "aws_s3_bucket" "object_bucket" {
    bucket = "tf-object-test-bucket-%d"
}
resource "aws_s3_bucket_object" "object" {
	bucket = "${aws_s3_bucket.object_bucket.bucket}"
	key = "test-key"
	source = "%s"
	content_type = "binary/octet-stream"
}
`, randomBucket, tf.Name())

var testAccAWSS3BucketObjectConfig_withContentCharacteristics = fmt.Sprintf(`
resource "aws_s3_bucket" "object_bucket_2" {
	bucket = "tf-object-test-bucket-%d"
}

resource "aws_s3_bucket_object" "object" {
	bucket = "${aws_s3_bucket.object_bucket_2.bucket}"
	key = "test-key"
	source = "%s"
	content_language = "en"
	content_type = "binary/octet-stream"
}
`, randomBucket, tf.Name())

var testAccAWSS3BucketObjectConfigContent = fmt.Sprintf(`
resource "aws_s3_bucket" "object_bucket" {
        bucket = "tf-object-test-bucket-%d"
}
resource "aws_s3_bucket_object" "object" {
        bucket = "${aws_s3_bucket.object_bucket.bucket}"
        key = "test-key"
        content = "some_bucket_content"
}
`, randomBucket)
