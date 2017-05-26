// make testacc TEST=./builtin/providers/aws/ TESTARGS='-run=TestAccDataSourceAWSS3BucketObject_'
package aws

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDataSourceAWSS3BucketObject_basic(t *testing.T) {
	rInt := acctest.RandInt()
	resourceOnlyConf, conf := testAccAWSDataSourceS3ObjectConfig_basic(rInt)

	var rObj s3.GetObjectOutput
	var dsObj s3.GetObjectOutput

	resource.Test(t, resource.TestCase{
		PreCheck:                  func() { testAccPreCheck(t) },
		Providers:                 testAccProviders,
		PreventPostDestroyRefresh: true,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: resourceOnlyConf,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSS3BucketObjectExists("aws_s3_bucket_object.object", &rObj),
				),
			},
			resource.TestStep{
				Config: conf,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsS3ObjectDataSourceExists("data.aws_s3_bucket_object.obj", &dsObj),
					resource.TestCheckResourceAttr("data.aws_s3_bucket_object.obj", "content_length", "11"),
					resource.TestCheckResourceAttr("data.aws_s3_bucket_object.obj", "content_type", "binary/octet-stream"),
					resource.TestCheckResourceAttr("data.aws_s3_bucket_object.obj", "etag", "b10a8db164e0754105b7a99be72e3fe5"),
					resource.TestMatchResourceAttr("data.aws_s3_bucket_object.obj", "last_modified",
						regexp.MustCompile("^[a-zA-Z]{3}, [0-9]+ [a-zA-Z]+ [0-9]{4} [0-9:]+ [A-Z]+$")),
					resource.TestCheckNoResourceAttr("data.aws_s3_bucket_object.obj", "body"),
				),
			},
		},
	})
}

func TestAccDataSourceAWSS3BucketObject_readableBody(t *testing.T) {
	rInt := acctest.RandInt()
	resourceOnlyConf, conf := testAccAWSDataSourceS3ObjectConfig_readableBody(rInt)

	var rObj s3.GetObjectOutput
	var dsObj s3.GetObjectOutput

	resource.Test(t, resource.TestCase{
		PreCheck:                  func() { testAccPreCheck(t) },
		Providers:                 testAccProviders,
		PreventPostDestroyRefresh: true,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: resourceOnlyConf,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSS3BucketObjectExists("aws_s3_bucket_object.object", &rObj),
				),
			},
			resource.TestStep{
				Config: conf,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsS3ObjectDataSourceExists("data.aws_s3_bucket_object.obj", &dsObj),
					resource.TestCheckResourceAttr("data.aws_s3_bucket_object.obj", "content_length", "3"),
					resource.TestCheckResourceAttr("data.aws_s3_bucket_object.obj", "content_type", "text/plain"),
					resource.TestCheckResourceAttr("data.aws_s3_bucket_object.obj", "etag", "a6105c0a611b41b08f1209506350279e"),
					resource.TestMatchResourceAttr("data.aws_s3_bucket_object.obj", "last_modified",
						regexp.MustCompile("^[a-zA-Z]{3}, [0-9]+ [a-zA-Z]+ [0-9]{4} [0-9:]+ [A-Z]+$")),
					resource.TestCheckResourceAttr("data.aws_s3_bucket_object.obj", "body", "yes"),
				),
			},
		},
	})
}

func TestAccDataSourceAWSS3BucketObject_kmsEncrypted(t *testing.T) {
	rInt := acctest.RandInt()
	resourceOnlyConf, conf := testAccAWSDataSourceS3ObjectConfig_kmsEncrypted(rInt)

	var rObj s3.GetObjectOutput
	var dsObj s3.GetObjectOutput

	resource.Test(t, resource.TestCase{
		PreCheck:                  func() { testAccPreCheck(t) },
		Providers:                 testAccProviders,
		PreventPostDestroyRefresh: true,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: resourceOnlyConf,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSS3BucketObjectExists("aws_s3_bucket_object.object", &rObj),
				),
			},
			resource.TestStep{
				Config: conf,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsS3ObjectDataSourceExists("data.aws_s3_bucket_object.obj", &dsObj),
					resource.TestCheckResourceAttr("data.aws_s3_bucket_object.obj", "content_length", "22"),
					resource.TestCheckResourceAttr("data.aws_s3_bucket_object.obj", "content_type", "text/plain"),
					resource.TestMatchResourceAttr("data.aws_s3_bucket_object.obj", "etag", regexp.MustCompile("^[a-f0-9]{32}$")),
					resource.TestCheckResourceAttr("data.aws_s3_bucket_object.obj", "server_side_encryption", "aws:kms"),
					resource.TestMatchResourceAttr("data.aws_s3_bucket_object.obj", "sse_kms_key_id",
						regexp.MustCompile("^arn:aws:kms:[a-z]{2}-[a-z]+-\\d{1}:[0-9]{12}:key/[a-z0-9-]{36}$")),
					resource.TestMatchResourceAttr("data.aws_s3_bucket_object.obj", "last_modified",
						regexp.MustCompile("^[a-zA-Z]{3}, [0-9]+ [a-zA-Z]+ [0-9]{4} [0-9:]+ [A-Z]+$")),
					resource.TestCheckResourceAttr("data.aws_s3_bucket_object.obj", "body", "Keep Calm and Carry On"),
				),
			},
		},
	})
}

func TestAccDataSourceAWSS3BucketObject_allParams(t *testing.T) {
	rInt := acctest.RandInt()
	resourceOnlyConf, conf := testAccAWSDataSourceS3ObjectConfig_allParams(rInt)

	var rObj s3.GetObjectOutput
	var dsObj s3.GetObjectOutput

	resource.Test(t, resource.TestCase{
		PreCheck:                  func() { testAccPreCheck(t) },
		Providers:                 testAccProviders,
		PreventPostDestroyRefresh: true,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: resourceOnlyConf,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSS3BucketObjectExists("aws_s3_bucket_object.object", &rObj),
				),
			},
			resource.TestStep{
				Config: conf,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsS3ObjectDataSourceExists("data.aws_s3_bucket_object.obj", &dsObj),
					resource.TestCheckResourceAttr("data.aws_s3_bucket_object.obj", "content_length", "21"),
					resource.TestCheckResourceAttr("data.aws_s3_bucket_object.obj", "content_type", "application/unknown"),
					resource.TestCheckResourceAttr("data.aws_s3_bucket_object.obj", "etag", "723f7a6ac0c57b445790914668f98640"),
					resource.TestMatchResourceAttr("data.aws_s3_bucket_object.obj", "last_modified",
						regexp.MustCompile("^[a-zA-Z]{3}, [0-9]+ [a-zA-Z]+ [0-9]{4} [0-9:]+ [A-Z]+$")),
					resource.TestMatchResourceAttr("data.aws_s3_bucket_object.obj", "version_id", regexp.MustCompile("^.{32}$")),
					resource.TestCheckNoResourceAttr("data.aws_s3_bucket_object.obj", "body"),
					resource.TestCheckResourceAttr("data.aws_s3_bucket_object.obj", "cache_control", "no-cache"),
					resource.TestCheckResourceAttr("data.aws_s3_bucket_object.obj", "content_disposition", "attachment"),
					resource.TestCheckResourceAttr("data.aws_s3_bucket_object.obj", "content_encoding", "identity"),
					resource.TestCheckResourceAttr("data.aws_s3_bucket_object.obj", "content_language", "en-GB"),
					// Encryption is off
					resource.TestCheckResourceAttr("data.aws_s3_bucket_object.obj", "server_side_encryption", ""),
					resource.TestCheckResourceAttr("data.aws_s3_bucket_object.obj", "sse_kms_key_id", ""),
					// Supported, but difficult to reproduce in short testing time
					resource.TestCheckResourceAttr("data.aws_s3_bucket_object.obj", "storage_class", "STANDARD"),
					resource.TestCheckResourceAttr("data.aws_s3_bucket_object.obj", "expiration", ""),
					// Currently unsupported in aws_s3_bucket_object resource
					resource.TestCheckResourceAttr("data.aws_s3_bucket_object.obj", "expires", ""),
					resource.TestCheckResourceAttr("data.aws_s3_bucket_object.obj", "website_redirect_location", ""),
					resource.TestCheckResourceAttr("data.aws_s3_bucket_object.obj", "metadata.%", "0"),
					resource.TestCheckResourceAttr("data.aws_s3_bucket_object.obj", "tags.%", "1"),
				),
			},
		},
	})
}

func testAccCheckAwsS3ObjectDataSourceExists(n string, obj *s3.GetObjectOutput) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Can't find S3 object data source: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("S3 object data source ID not set")
		}

		s3conn := testAccProvider.Meta().(*AWSClient).s3conn
		out, err := s3conn.GetObject(
			&s3.GetObjectInput{
				Bucket: aws.String(rs.Primary.Attributes["bucket"]),
				Key:    aws.String(rs.Primary.Attributes["key"]),
			})
		if err != nil {
			return fmt.Errorf("Failed getting S3 Object from %s: %s",
				rs.Primary.Attributes["bucket"]+"/"+rs.Primary.Attributes["key"], err)
		}

		*obj = *out

		return nil
	}
}

func testAccAWSDataSourceS3ObjectConfig_basic(randInt int) (string, string) {
	resources := fmt.Sprintf(`
resource "aws_s3_bucket" "object_bucket" {
	bucket = "tf-object-test-bucket-%d"
}
resource "aws_s3_bucket_object" "object" {
	bucket = "${aws_s3_bucket.object_bucket.bucket}"
	key = "tf-testing-obj-%d"
	content = "Hello World"
}
`, randInt, randInt)

	both := fmt.Sprintf(`%s
data "aws_s3_bucket_object" "obj" {
	bucket = "tf-object-test-bucket-%d"
	key = "tf-testing-obj-%d"
}`, resources, randInt, randInt)

	return resources, both
}

func testAccAWSDataSourceS3ObjectConfig_readableBody(randInt int) (string, string) {
	resources := fmt.Sprintf(`
resource "aws_s3_bucket" "object_bucket" {
	bucket = "tf-object-test-bucket-%d"
}
resource "aws_s3_bucket_object" "object" {
	bucket = "${aws_s3_bucket.object_bucket.bucket}"
	key = "tf-testing-obj-%d-readable"
	content = "yes"
	content_type = "text/plain"
}
`, randInt, randInt)

	both := fmt.Sprintf(`%s
data "aws_s3_bucket_object" "obj" {
	bucket = "tf-object-test-bucket-%d"
	key = "tf-testing-obj-%d-readable"
}`, resources, randInt, randInt)

	return resources, both
}

func testAccAWSDataSourceS3ObjectConfig_kmsEncrypted(randInt int) (string, string) {
	resources := fmt.Sprintf(`
resource "aws_s3_bucket" "object_bucket" {
	bucket = "tf-object-test-bucket-%d"
}
resource "aws_kms_key" "example" {
  description             = "TF Acceptance Test KMS key"
  deletion_window_in_days = 7
}
resource "aws_s3_bucket_object" "object" {
	bucket = "${aws_s3_bucket.object_bucket.bucket}"
	key = "tf-testing-obj-%d-encrypted"
	content = "Keep Calm and Carry On"
	content_type = "text/plain"
	kms_key_id = "${aws_kms_key.example.arn}"
}
`, randInt, randInt)

	both := fmt.Sprintf(`%s
data "aws_s3_bucket_object" "obj" {
	bucket = "tf-object-test-bucket-%d"
	key = "tf-testing-obj-%d-encrypted"
}`, resources, randInt, randInt)

	return resources, both
}

func testAccAWSDataSourceS3ObjectConfig_allParams(randInt int) (string, string) {
	resources := fmt.Sprintf(`
resource "aws_s3_bucket" "object_bucket" {
	bucket = "tf-object-test-bucket-%d"
	versioning {
		enabled = true
	}
}

resource "aws_s3_bucket_object" "object" {
	bucket = "${aws_s3_bucket.object_bucket.bucket}"
	key = "tf-testing-obj-%d-all-params"
	content = <<CONTENT
{"msg": "Hi there!"}
CONTENT
	content_type = "application/unknown"
	cache_control = "no-cache"
	content_disposition = "attachment"
	content_encoding = "identity"
	content_language = "en-GB"
	tags {
		Key1 = "Value 1"
	}
}
`, randInt, randInt)

	both := fmt.Sprintf(`%s
data "aws_s3_bucket_object" "obj" {
	bucket = "tf-object-test-bucket-%d"
	key = "tf-testing-obj-%d-all-params"
}`, resources, randInt, randInt)

	return resources, both
}
