package alicloud

import (
	"fmt"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"testing"
)

func TestAccAlicloudOssBucketObject_source(t *testing.T) {
	tmpFile, err := ioutil.TempFile("", "tf-oss-object-test-acc-source")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	// first write some data to the tempfile just so it's not 0 bytes.
	err = ioutil.WriteFile(tmpFile.Name(), []byte("{anything will do }"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	var obj http.Header
	bucket := fmt.Sprintf("tf-oss-test-object-source-%d", acctest.RandInt())
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAlicloudOssBucketObjectDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(`
						resource "alicloud_oss_bucket" "bucket" {
						    bucket = "%s"
						}
						resource "alicloud_oss_bucket_object" "source" {
							bucket = "${alicloud_oss_bucket.bucket.bucket}"
							key = "test-object-source-key"
							source = "%s"
							content_type = "binary/octet-stream"
						}`, bucket, tmpFile.Name()),
				Check: testAccCheckAlicloudOssBucketObjectExists(
					"alicloud_oss_bucket_object.source", bucket, obj),
			},
		},
	})
}

func TestAccAlicloudOssBucketObject_content(t *testing.T) {
	var obj http.Header
	bucket := fmt.Sprintf("tf-object-test-object-content-%d", acctest.RandInt())
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAlicloudOssBucketObjectDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(`
						resource "alicloud_oss_bucket" "bucket" {
						    bucket = "%s"
						}
						resource "alicloud_oss_bucket_object" "content" {
							bucket = "${alicloud_oss_bucket.bucket.bucket}"
							key = "test-object-content-key"
							content = "some words for test oss object content"
						}`, bucket),
				Check: testAccCheckAlicloudOssBucketObjectExists(
					"alicloud_oss_bucket_object.content", bucket, obj),
			},
		},
	})
}

func TestAccAlicloudOssBucketObject_acl(t *testing.T) {
	tmpFile, err := ioutil.TempFile("", "tf-oss-object-test-acc-source")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	// first write some data to the tempfile just so it's not 0 bytes.
	err = ioutil.WriteFile(tmpFile.Name(), []byte("{anything will do }"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	var obj http.Header
	bucket := fmt.Sprintf("tf-object-test-bucket-%d", acctest.RandInt())
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAlicloudOssBucketObjectDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(`
						resource "alicloud_oss_bucket" "bucket" {
							bucket = "%s"
						}

						resource "alicloud_oss_bucket_object" "acl" {
							bucket = "${alicloud_oss_bucket.bucket.bucket}"
							key = "test-object-acl-key"
							source = "%s"
							acl = "%s"
						}
						`, bucket, tmpFile.Name(), "public-read"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAlicloudOssBucketObjectExists(
						"alicloud_oss_bucket_object.acl", bucket, obj),
					resource.TestCheckResourceAttr(
						"alicloud_oss_bucket_object.acl",
						"acl",
						"public-read"),
				),
			},
		},
	})
}

func testAccCheckAlicloudOssBucketObjectExists(n string, bucket string, obj http.Header) resource.TestCheckFunc {
	providers := []*schema.Provider{testAccProvider}
	return testAccCheckOssBucketObjectExistsWithProviders(n, bucket, obj, &providers)
}
func testAccCheckOssBucketObjectExistsWithProviders(n string, bucket string, obj http.Header, providers *[]*schema.Provider) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}
		for _, provider := range *providers {
			// Ignore if Meta is empty, this can happen for validation providers
			if provider.Meta() == nil {
				continue
			}
			client, err := provider.Meta().(*AliyunClient).ossconn.Bucket(bucket)
			if err != nil {
				return fmt.Errorf("Error getting bucket: %#v", err)
			}
			object, err := client.GetObjectMeta(rs.Primary.ID)
			log.Printf("[WARN]get oss bucket object %#v", bucket)
			if err == nil {
				if object != nil {
					obj = object
					return nil
				}
				continue
			} else if err != nil {
				return err

			}
		}

		return fmt.Errorf("Bucket not found")
	}
}
func testAccCheckAlicloudOssBucketObjectDestroy(s *terraform.State) error {
	return testAccCheckOssBucketObjectDestroyWithProvider(s, testAccProvider)
}

func testAccCheckOssBucketObjectDestroyWithProvider(s *terraform.State, provider *schema.Provider) error {
	client := provider.Meta().(*AliyunClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "alicloud_oss_bucket" {
			continue
		}

		// Try to find the resource
		bucket, err := client.QueryOssBucketById(rs.Primary.ID)
		if err == nil {
			if bucket.Name != "" {
				return fmt.Errorf("Found instance: %s", bucket.Name)
			}
		}

		// Verify the error is what we want
		e, _ := err.(oss.ServiceError)
		if e.Code == OssBucketNotFound {
			continue
		}

		return err
	}

	return nil
}
