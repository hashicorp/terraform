package google

import (
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"google.golang.org/api/storage/v1"
)

var tf, err = ioutil.TempFile("", "tf-gce-test")
var bucketName = "tf-gce-bucket-test"
var objectName = "tf-gce-test"
var content = "now this is content!"

func TestAccGoogleStorageObject_basic(t *testing.T) {
	bucketName := testBucketName()
	data := []byte("data data data")
	h := md5.New()
	h.Write(data)
	data_md5 := base64.StdEncoding.EncodeToString(h.Sum(nil))

	ioutil.WriteFile(tf.Name(), data, 0644)
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			if err != nil {
				panic(err)
			}
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccGoogleStorageObjectDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testGoogleStorageBucketsObjectBasic(bucketName),
				Check:  testAccCheckGoogleStorageObject(bucketName, objectName, data_md5),
			},
		},
	})
}

func TestAccGoogleStorageObject_content(t *testing.T) {
	bucketName := testBucketName()
	data := []byte(content)
	h := md5.New()
	h.Write(data)
	data_md5 := base64.StdEncoding.EncodeToString(h.Sum(nil))

	ioutil.WriteFile(tf.Name(), data, 0644)
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			if err != nil {
				panic(err)
			}
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccGoogleStorageObjectDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testGoogleStorageBucketsObjectContent(bucketName),
				Check:  testAccCheckGoogleStorageObject(bucketName, objectName, data_md5),
			},
		},
	})
}

func testAccCheckGoogleStorageObject(bucket, object, md5 string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		config := testAccProvider.Meta().(*Config)

		objectsService := storage.NewObjectsService(config.clientStorage)

		getCall := objectsService.Get(bucket, object)
		res, err := getCall.Do()

		if err != nil {
			return fmt.Errorf("Error retrieving contents of object %s: %s", object, err)
		}

		if md5 != res.Md5Hash {
			return fmt.Errorf("Error contents of %s garbled, md5 hashes don't match (%s, %s)", object, md5, res.Md5Hash)
		}

		return nil
	}
}

func testAccGoogleStorageObjectDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "google_storage_bucket_object" {
			continue
		}

		bucket := rs.Primary.Attributes["bucket"]
		name := rs.Primary.Attributes["name"]

		objectsService := storage.NewObjectsService(config.clientStorage)

		getCall := objectsService.Get(bucket, name)
		_, err := getCall.Do()

		if err == nil {
			return fmt.Errorf("Object %s still exists", name)
		}
	}

	return nil
}

func testGoogleStorageBucketsObjectContent(bucketName string) string {
	return fmt.Sprintf(`
resource "google_storage_bucket" "bucket" {
	name = "%s"
}

resource "google_storage_bucket_object" "object" {
	name = "%s"
	bucket = "${google_storage_bucket.bucket.name}"
	content = "%s"
	predefined_acl = "projectPrivate"
}
`, bucketName, objectName, content)
}
func testGoogleStorageBucketsObjectBasic(bucketName string) string {
	return fmt.Sprintf(`
resource "google_storage_bucket" "bucket" {
	name = "%s"
}

resource "google_storage_bucket_object" "object" {
	name = "%s"
	bucket = "${google_storage_bucket.bucket.name}"
	source = "%s"
	predefined_acl = "projectPrivate"
}
`, bucketName, objectName, tf.Name())
}
