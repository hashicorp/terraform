// make testacc TEST=./builtin/providers/aws/ TESTARGS='-run=TestAccAWSS3BucketObject_'
package aws

import (
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"sort"
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
	var obj s3.GetObjectOutput

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSS3BucketObjectDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSS3BucketObjectConfigSource(rInt, tmpFile.Name()),
				Check:  testAccCheckAWSS3BucketObjectExists("aws_s3_bucket_object.object", &obj),
			},
		},
	})
}

func TestAccAWSS3BucketObject_content(t *testing.T) {
	rInt := acctest.RandInt()
	var obj s3.GetObjectOutput

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSS3BucketObjectDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				PreConfig: func() {},
				Config:    testAccAWSS3BucketObjectConfigContent(rInt),
				Check:     testAccCheckAWSS3BucketObjectExists("aws_s3_bucket_object.object", &obj),
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

	var obj s3.GetObjectOutput

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSS3BucketObjectDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSS3BucketObjectConfig_withContentCharacteristics(rInt, tmpFile.Name()),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSS3BucketObjectExists("aws_s3_bucket_object.object", &obj),
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
	var obj s3.GetObjectOutput

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSS3BucketObjectDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSS3BucketObjectConfig_updates(rInt, tmpFile.Name()),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSS3BucketObjectExists("aws_s3_bucket_object.object", &obj),
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
					testAccCheckAWSS3BucketObjectExists("aws_s3_bucket_object.object", &obj),
					resource.TestCheckResourceAttr("aws_s3_bucket_object.object", "etag", "1c7fd13df1515c2a13ad9eb068931f09"),
				),
			},
		},
	})
}

func TestAccAWSS3BucketObject_updatesWithVersioning(t *testing.T) {
	tmpFile, err := ioutil.TempFile("", "tf-acc-s3-obj-updates-w-versions")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	rInt := acctest.RandInt()
	err = ioutil.WriteFile(tmpFile.Name(), []byte("initial versioned object state"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	var originalObj, modifiedObj s3.GetObjectOutput

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSS3BucketObjectDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSS3BucketObjectConfig_updatesWithVersioning(rInt, tmpFile.Name()),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSS3BucketObjectExists("aws_s3_bucket_object.object", &originalObj),
					resource.TestCheckResourceAttr("aws_s3_bucket_object.object", "etag", "cee4407fa91906284e2a5e5e03e86b1b"),
				),
			},
			resource.TestStep{
				PreConfig: func() {
					err = ioutil.WriteFile(tmpFile.Name(), []byte("modified versioned object"), 0644)
					if err != nil {
						t.Fatal(err)
					}
				},
				Config: testAccAWSS3BucketObjectConfig_updatesWithVersioning(rInt, tmpFile.Name()),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSS3BucketObjectExists("aws_s3_bucket_object.object", &modifiedObj),
					resource.TestCheckResourceAttr("aws_s3_bucket_object.object", "etag", "00b8c73b1b50e7cc932362c7225b8e29"),
					testAccCheckAWSS3BucketObjectVersionIdDiffers(&originalObj, &modifiedObj),
				),
			},
		},
	})
}

func testAccCheckAWSS3BucketObjectVersionIdDiffers(first, second *s3.GetObjectOutput) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if first.VersionId == nil {
			return fmt.Errorf("Expected first object to have VersionId: %s", first)
		}
		if second.VersionId == nil {
			return fmt.Errorf("Expected second object to have VersionId: %s", second)
		}

		if *first.VersionId == *second.VersionId {
			return fmt.Errorf("Expected Version IDs to differ, but they are equal (%s)", *first.VersionId)
		}

		return nil
	}
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

func testAccCheckAWSS3BucketObjectExists(n string, obj *s3.GetObjectOutput) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not Found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No S3 Bucket Object ID is set")
		}

		s3conn := testAccProvider.Meta().(*AWSClient).s3conn
		out, err := s3conn.GetObject(
			&s3.GetObjectInput{
				Bucket:  aws.String(rs.Primary.Attributes["bucket"]),
				Key:     aws.String(rs.Primary.Attributes["key"]),
				IfMatch: aws.String(rs.Primary.Attributes["etag"]),
			})
		if err != nil {
			return fmt.Errorf("S3Bucket Object error: %s", err)
		}

		*obj = *out

		return nil
	}
}

func TestAccAWSS3BucketObject_kms(t *testing.T) {
	rInt := acctest.RandInt()
	var obj s3.GetObjectOutput

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSS3BucketObjectDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				PreConfig: func() {},
				Config:    testAccAWSS3BucketObjectConfig_withKMSId(rInt),
				Check:     testAccCheckAWSS3BucketObjectExists("aws_s3_bucket_object.object", &obj),
			},
		},
	})
}

func TestAccAWSS3BucketObject_sse(t *testing.T) {
	tmpFile, err := ioutil.TempFile("", "tf-acc-s3-obj-source-sse")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	// first write some data to the tempfile just so it's not 0 bytes.
	err = ioutil.WriteFile(tmpFile.Name(), []byte("{anything will do}"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	rInt := acctest.RandInt()
	var obj s3.GetObjectOutput

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSS3BucketObjectDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				PreConfig: func() {},
				Config:    testAccAWSS3BucketObjectConfig_withSSE(rInt, tmpFile.Name()),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSS3BucketObjectExists(
						"aws_s3_bucket_object.object",
						&obj),
					testAccCheckAWSS3BucketObjectSSE(
						"aws_s3_bucket_object.object",
						"aws:kms"),
				),
			},
		},
	})
}

func TestAccAWSS3BucketObject_acl(t *testing.T) {
	rInt := acctest.RandInt()
	var obj s3.GetObjectOutput

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSS3BucketObjectDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSS3BucketObjectConfig_acl(rInt, "private"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSS3BucketObjectExists(
						"aws_s3_bucket_object.object", &obj),
					resource.TestCheckResourceAttr(
						"aws_s3_bucket_object.object",
						"acl",
						"private"),
					testAccCheckAWSS3BucketObjectAcl(
						"aws_s3_bucket_object.object",
						[]string{"FULL_CONTROL"}),
				),
			},
			resource.TestStep{
				Config: testAccAWSS3BucketObjectConfig_acl(rInt, "public-read"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSS3BucketObjectExists(
						"aws_s3_bucket_object.object",
						&obj),
					resource.TestCheckResourceAttr(
						"aws_s3_bucket_object.object",
						"acl",
						"public-read"),
					testAccCheckAWSS3BucketObjectAcl(
						"aws_s3_bucket_object.object",
						[]string{"FULL_CONTROL", "READ"}),
				),
			},
		},
	})
}

func testAccCheckAWSS3BucketObjectAcl(n string, expectedPerms []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, _ := s.RootModule().Resources[n]
		s3conn := testAccProvider.Meta().(*AWSClient).s3conn

		out, err := s3conn.GetObjectAcl(&s3.GetObjectAclInput{
			Bucket: aws.String(rs.Primary.Attributes["bucket"]),
			Key:    aws.String(rs.Primary.Attributes["key"]),
		})

		if err != nil {
			return fmt.Errorf("GetObjectAcl error: %v", err)
		}

		var perms []string
		for _, v := range out.Grants {
			perms = append(perms, *v.Permission)
		}
		sort.Strings(perms)

		if !reflect.DeepEqual(perms, expectedPerms) {
			return fmt.Errorf("Expected ACL permissions to be %v, got %v", expectedPerms, perms)
		}

		return nil
	}
}

func TestResourceAWSS3BucketObjectAcl_validation(t *testing.T) {
	_, errors := validateS3BucketObjectAclType("incorrect", "acl")
	if len(errors) == 0 {
		t.Fatalf("Expected to trigger a validation error")
	}

	var testCases = []struct {
		Value    string
		ErrCount int
	}{
		{
			Value:    "public-read",
			ErrCount: 0,
		},
		{
			Value:    "public-read-write",
			ErrCount: 0,
		},
	}

	for _, tc := range testCases {
		_, errors := validateS3BucketObjectAclType(tc.Value, "acl")
		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected not to trigger a validation error")
		}
	}
}

func TestAccAWSS3BucketObject_storageClass(t *testing.T) {
	rInt := acctest.RandInt()
	var obj s3.GetObjectOutput

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSS3BucketObjectDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				PreConfig: func() {},
				Config:    testAccAWSS3BucketObjectConfigContent(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSS3BucketObjectExists(
						"aws_s3_bucket_object.object",
						&obj),
					resource.TestCheckResourceAttr(
						"aws_s3_bucket_object.object",
						"storage_class",
						"STANDARD"),
					testAccCheckAWSS3BucketObjectStorageClass(
						"aws_s3_bucket_object.object",
						"STANDARD"),
				),
			},
			resource.TestStep{
				Config: testAccAWSS3BucketObjectConfig_storageClass(rInt, "REDUCED_REDUNDANCY"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSS3BucketObjectExists(
						"aws_s3_bucket_object.object",
						&obj),
					resource.TestCheckResourceAttr(
						"aws_s3_bucket_object.object",
						"storage_class",
						"REDUCED_REDUNDANCY"),
					testAccCheckAWSS3BucketObjectStorageClass(
						"aws_s3_bucket_object.object",
						"REDUCED_REDUNDANCY"),
				),
			},
		},
	})
}

func TestResourceAWSS3BucketObjectStorageClass_validation(t *testing.T) {
	_, errors := validateS3BucketObjectStorageClassType("incorrect", "storage_class")
	if len(errors) == 0 {
		t.Fatalf("Expected to trigger a validation error")
	}

	var testCases = []struct {
		Value    string
		ErrCount int
	}{
		{
			Value:    "STANDARD",
			ErrCount: 0,
		},
		{
			Value:    "REDUCED_REDUNDANCY",
			ErrCount: 0,
		},
	}

	for _, tc := range testCases {
		_, errors := validateS3BucketObjectStorageClassType(tc.Value, "storage_class")
		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected not to trigger a validation error")
		}
	}
}

func testAccCheckAWSS3BucketObjectStorageClass(n, expectedClass string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, _ := s.RootModule().Resources[n]
		s3conn := testAccProvider.Meta().(*AWSClient).s3conn

		out, err := s3conn.HeadObject(&s3.HeadObjectInput{
			Bucket: aws.String(rs.Primary.Attributes["bucket"]),
			Key:    aws.String(rs.Primary.Attributes["key"]),
		})

		if err != nil {
			return fmt.Errorf("HeadObject error: %v", err)
		}

		// The "STANDARD" (which is also the default) storage
		// class when set would not be included in the results.
		storageClass := s3.StorageClassStandard
		if out.StorageClass != nil {
			storageClass = *out.StorageClass
		}

		if storageClass != expectedClass {
			return fmt.Errorf("Expected Storage Class to be %v, got %v",
				expectedClass, storageClass)
		}

		return nil
	}
}

func testAccCheckAWSS3BucketObjectSSE(n, expectedSSE string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, _ := s.RootModule().Resources[n]
		s3conn := testAccProvider.Meta().(*AWSClient).s3conn

		out, err := s3conn.HeadObject(&s3.HeadObjectInput{
			Bucket: aws.String(rs.Primary.Attributes["bucket"]),
			Key:    aws.String(rs.Primary.Attributes["key"]),
		})

		if err != nil {
			return fmt.Errorf("HeadObject error: %v", err)
		}

		if out.ServerSideEncryption == nil {
			return fmt.Errorf("Expected a non %v Server Side Encryption.", out.ServerSideEncryption)
		}

		sse := *out.ServerSideEncryption
		if sse != expectedSSE {
			return fmt.Errorf("Expected Server Side Encryption %v, got %v.",
				expectedSSE, sse)
		}

		return nil
	}
}

func TestAccAWSS3BucketObject_tags(t *testing.T) {
	rInt := acctest.RandInt()
	var obj s3.GetObjectOutput

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSS3BucketObjectDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				PreConfig: func() {},
				Config:    testAccAWSS3BucketObjectConfig_withTags(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSS3BucketObjectExists("aws_s3_bucket_object.object", &obj),
					resource.TestCheckResourceAttr("aws_s3_bucket_object.object", "tags.%", "2"),
				),
			},
		},
	})
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

func testAccAWSS3BucketObjectConfig_updatesWithVersioning(randInt int, source string) string {
	return fmt.Sprintf(`
resource "aws_s3_bucket" "object_bucket_3" {
	bucket = "tf-object-test-bucket-%d"
	versioning {
		enabled = true
	}
}

resource "aws_s3_bucket_object" "object" {
	bucket = "${aws_s3_bucket.object_bucket_3.bucket}"
	key = "updateable-key"
	source = "%s"
	etag = "${md5(file("%s"))}"
}
`, randInt, source, source)
}

func testAccAWSS3BucketObjectConfig_withKMSId(randInt int) string {
	return fmt.Sprintf(`
resource "aws_kms_key" "kms_key_1" {
}

resource "aws_s3_bucket" "object_bucket_2" {
	bucket = "tf-object-test-bucket-%d"
}

resource "aws_s3_bucket_object" "object" {
	bucket = "${aws_s3_bucket.object_bucket_2.bucket}"
	key = "test-key"
	content = "stuff"
	kms_key_id = "${aws_kms_key.kms_key_1.arn}"
}
`, randInt)
}

func testAccAWSS3BucketObjectConfig_withSSE(randInt int, source string) string {
	return fmt.Sprintf(`
resource "aws_s3_bucket" "object_bucket" {
	bucket = "tf-object-test-bucket-%d"
}

resource "aws_s3_bucket_object" "object" {
	bucket = "${aws_s3_bucket.object_bucket.bucket}"
	key = "test-key"
	source = "%s"
	server_side_encryption = "aws:kms"
}
`, randInt, source)
}

func testAccAWSS3BucketObjectConfig_acl(randInt int, acl string) string {
	return fmt.Sprintf(`
resource "aws_s3_bucket" "object_bucket" {
        bucket = "tf-object-test-bucket-%d"
}
resource "aws_s3_bucket_object" "object" {
        bucket = "${aws_s3_bucket.object_bucket.bucket}"
        key = "test-key"
        content = "some_bucket_content"
        acl = "%s"
}
`, randInt, acl)
}

func testAccAWSS3BucketObjectConfig_storageClass(randInt int, storage_class string) string {
	return fmt.Sprintf(`
resource "aws_s3_bucket" "object_bucket" {
        bucket = "tf-object-test-bucket-%d"
}
resource "aws_s3_bucket_object" "object" {
        bucket = "${aws_s3_bucket.object_bucket.bucket}"
        key = "test-key"
        content = "some_bucket_content"
        storage_class = "%s"
}
`, randInt, storage_class)
}

func testAccAWSS3BucketObjectConfig_withTags(randInt int) string {
	return fmt.Sprintf(`
resource "aws_s3_bucket" "object_bucket_2" {
	bucket = "tf-object-test-bucket-%d"
}

resource "aws_s3_bucket_object" "object" {
	bucket = "${aws_s3_bucket.object_bucket_2.bucket}"
	key = "test-key"
	content = "stuff"
	tags {
		Key1 = "Value One"
		Description = "Very interesting"
	}
}
`, randInt)
}
