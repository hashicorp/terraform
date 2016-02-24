package aws

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

func TestAccAWSS3BucketObject_source(t *testing.T) {
	tmpFile, err := ioutil.TempFile("", "tf-acc-s3-obj-source")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	rInt := acctest.RandInt()
	// first write some data to the tempfile just so it's not 0 bytes.
	err = ioutil.WriteFile(tmpFile.Name(), []byte("{anything will do }"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSS3BucketObjectDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSS3BucketObjectConfigSource(rInt, tmpFile.Name()),
				Check:  testAccCheckAWSS3BucketObjectExists("aws_s3_bucket_object.object"),
			},
		},
	})
}

func TestAccAWSS3BucketObject_content(t *testing.T) {
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSS3BucketObjectDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				PreConfig: func() {},
				Config:    testAccAWSS3BucketObjectConfigContent(rInt),
				Check:     testAccCheckAWSS3BucketObjectExists("aws_s3_bucket_object.object"),
			},
		},
	})
}

func TestAccAWSS3BucketObject_withContentCharacteristics(t *testing.T) {
	tmpFile, err := ioutil.TempFile("", "tf-acc-s3-obj-content-characteristics")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	rInt := acctest.RandInt()
	// first write some data to the tempfile just so it's not 0 bytes.
	err = ioutil.WriteFile(tmpFile.Name(), []byte("{anything will do }"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSS3BucketObjectDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSS3BucketObjectConfig_withContentCharacteristics(rInt, tmpFile.Name()),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSS3BucketObjectExists("aws_s3_bucket_object.object"),
					resource.TestCheckResourceAttr(
						"aws_s3_bucket_object.object", "content_type", "binary/octet-stream"),
				),
			},
		},
	})
}

func TestAccAWSS3BucketObject_updates(t *testing.T) {
	tmpFile, err := ioutil.TempFile("", "tf-acc-s3-obj-updates")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	rInt := acctest.RandInt()
	err = ioutil.WriteFile(tmpFile.Name(), []byte("initial object state"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSS3BucketObjectDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSS3BucketObjectConfig_updates(rInt, tmpFile.Name()),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSS3BucketObjectExists("aws_s3_bucket_object.object"),
					resource.TestCheckResourceAttr("aws_s3_bucket_object.object", "etag", "647d1d58e1011c743ec67d5e8af87b53"),
				),
			},
			resource.TestStep{
				PreConfig: func() {
					err = ioutil.WriteFile(tmpFile.Name(), []byte("modified object"), 0644)
					if err != nil {
						t.Fatal(err)
					}
				},
				Config: testAccAWSS3BucketObjectConfig_updates(rInt, tmpFile.Name()),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSS3BucketObjectExists("aws_s3_bucket_object.object"),
					resource.TestCheckResourceAttr("aws_s3_bucket_object.object", "etag", "1c7fd13df1515c2a13ad9eb068931f09"),
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

func testAccAWSS3BucketObjectConfigSource(randInt int, source string) string {
	return fmt.Sprintf(`
resource "aws_s3_bucket" "object_bucket" {
    bucket = "tf-object-test-bucket-%d"
}
resource "aws_s3_bucket_object" "object" {
	bucket = "${aws_s3_bucket.object_bucket.bucket}"
	key = "test-key"
	source = "%s"
	content_type = "binary/octet-stream"
}
`, randInt, source)
}

func testAccAWSS3BucketObjectConfig_withContentCharacteristics(randInt int, source string) string {
	return fmt.Sprintf(`
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
`, randInt, source)
}

func testAccAWSS3BucketObjectConfigContent(randInt int) string {
	return fmt.Sprintf(`
resource "aws_s3_bucket" "object_bucket" {
        bucket = "tf-object-test-bucket-%d"
}
resource "aws_s3_bucket_object" "object" {
        bucket = "${aws_s3_bucket.object_bucket.bucket}"
        key = "test-key"
        content = "some_bucket_content"
}
`, randInt)
}

func testAccAWSS3BucketObjectConfig_updates(randInt int, source string) string {
	return fmt.Sprintf(`
resource "aws_s3_bucket" "object_bucket_3" {
	bucket = "tf-object-test-bucket-%d"
}

resource "aws_s3_bucket_object" "object" {
	bucket = "${aws_s3_bucket.object_bucket_3.bucket}"
	key = "updateable-key"
	source = "%s"
	etag = "${md5(file("%s"))}"
}
`, randInt, source, source)
}
