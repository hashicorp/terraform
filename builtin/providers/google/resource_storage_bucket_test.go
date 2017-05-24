package google

import (
	"bytes"
	"fmt"
	"log"
	"testing"

	"github.com/r3labs/terraform/helper/acctest"
	"github.com/r3labs/terraform/helper/resource"
	"github.com/r3labs/terraform/terraform"

	"google.golang.org/api/googleapi"
	storage "google.golang.org/api/storage/v1"
)

func TestAccStorageBucket_basic(t *testing.T) {
	var bucket storage.Bucket
	bucketName := fmt.Sprintf("tf-test-acl-bucket-%d", acctest.RandInt())

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccStorageBucketDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccStorageBucket_basic(bucketName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckStorageBucketExists(
						"google_storage_bucket.bucket", bucketName, &bucket),
					resource.TestCheckResourceAttr(
						"google_storage_bucket.bucket", "location", "US"),
					resource.TestCheckResourceAttr(
						"google_storage_bucket.bucket", "force_destroy", "false"),
				),
			},
		},
	})
}

func TestAccStorageBucket_customAttributes(t *testing.T) {
	var bucket storage.Bucket
	bucketName := fmt.Sprintf("tf-test-acl-bucket-%d", acctest.RandInt())

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccStorageBucketDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccStorageBucket_customAttributes(bucketName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckStorageBucketExists(
						"google_storage_bucket.bucket", bucketName, &bucket),
					resource.TestCheckResourceAttr(
						"google_storage_bucket.bucket", "location", "EU"),
					resource.TestCheckResourceAttr(
						"google_storage_bucket.bucket", "force_destroy", "true"),
				),
			},
		},
	})
}

func TestAccStorageBucket_storageClass(t *testing.T) {
	var bucket storage.Bucket
	bucketName := fmt.Sprintf("tf-test-acc-bucket-%d", acctest.RandInt())

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccStorageBucketDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccStorageBucket_storageClass(bucketName, "MULTI_REGIONAL", ""),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckStorageBucketExists(
						"google_storage_bucket.bucket", bucketName, &bucket),
					resource.TestCheckResourceAttr(
						"google_storage_bucket.bucket", "storage_class", "MULTI_REGIONAL"),
				),
			},
			{
				Config: testAccStorageBucket_storageClass(bucketName, "NEARLINE", ""),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckStorageBucketExists(
						"google_storage_bucket.bucket", bucketName, &bucket),
					resource.TestCheckResourceAttr(
						"google_storage_bucket.bucket", "storage_class", "NEARLINE"),
				),
			},
			{
				Config: testAccStorageBucket_storageClass(bucketName, "REGIONAL", "US-CENTRAL1"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckStorageBucketExists(
						"google_storage_bucket.bucket", bucketName, &bucket),
					resource.TestCheckResourceAttr(
						"google_storage_bucket.bucket", "storage_class", "REGIONAL"),
					resource.TestCheckResourceAttr(
						"google_storage_bucket.bucket", "location", "US-CENTRAL1"),
				),
			},
		},
	})
}

func TestAccStorageBucket_update(t *testing.T) {
	var bucket storage.Bucket
	bucketName := fmt.Sprintf("tf-test-acl-bucket-%d", acctest.RandInt())

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccStorageBucketDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccStorageBucket_basic(bucketName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckStorageBucketExists(
						"google_storage_bucket.bucket", bucketName, &bucket),
					resource.TestCheckResourceAttr(
						"google_storage_bucket.bucket", "location", "US"),
					resource.TestCheckResourceAttr(
						"google_storage_bucket.bucket", "force_destroy", "false"),
				),
			},
			resource.TestStep{
				Config: testAccStorageBucket_customAttributes(bucketName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckStorageBucketExists(
						"google_storage_bucket.bucket", bucketName, &bucket),
					resource.TestCheckResourceAttr(
						"google_storage_bucket.bucket", "predefined_acl", "publicReadWrite"),
					resource.TestCheckResourceAttr(
						"google_storage_bucket.bucket", "location", "EU"),
					resource.TestCheckResourceAttr(
						"google_storage_bucket.bucket", "force_destroy", "true"),
				),
			},
		},
	})
}

func TestAccStorageBucket_forceDestroy(t *testing.T) {
	var bucket storage.Bucket
	bucketName := fmt.Sprintf("tf-test-acl-bucket-%d", acctest.RandInt())

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccStorageBucketDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccStorageBucket_customAttributes(bucketName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckStorageBucketExists(
						"google_storage_bucket.bucket", bucketName, &bucket),
				),
			},
			resource.TestStep{
				Config: testAccStorageBucket_customAttributes(bucketName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckStorageBucketPutItem(bucketName),
				),
			},
			resource.TestStep{
				Config: testAccStorageBucket_customAttributes("idontexist"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckStorageBucketMissing(bucketName),
				),
			},
		},
	})
}

func testAccCheckStorageBucketExists(n string, bucketName string, bucket *storage.Bucket) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Project_ID is set")
		}

		config := testAccProvider.Meta().(*Config)

		found, err := config.clientStorage.Buckets.Get(rs.Primary.ID).Do()
		if err != nil {
			return err
		}

		if found.Id != rs.Primary.ID {
			return fmt.Errorf("Bucket not found")
		}

		if found.Name != bucketName {
			return fmt.Errorf("expected name %s, got %s", bucketName, found.Name)
		}

		*bucket = *found
		return nil
	}
}

func testAccCheckStorageBucketPutItem(bucketName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		config := testAccProvider.Meta().(*Config)

		data := bytes.NewBufferString("test")
		dataReader := bytes.NewReader(data.Bytes())
		object := &storage.Object{Name: "bucketDestroyTestFile"}

		// This needs to use Media(io.Reader) call, otherwise it does not go to /upload API and fails
		if res, err := config.clientStorage.Objects.Insert(bucketName, object).Media(dataReader).Do(); err == nil {
			log.Printf("[INFO] Created object %v at location %v\n\n", res.Name, res.SelfLink)
		} else {
			return fmt.Errorf("Objects.Insert failed: %v", err)
		}

		return nil
	}
}

func testAccCheckStorageBucketMissing(bucketName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		config := testAccProvider.Meta().(*Config)

		_, err := config.clientStorage.Buckets.Get(bucketName).Do()
		if err == nil {
			return fmt.Errorf("Found %s", bucketName)
		}

		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
			return nil
		}

		return err
	}
}

func testAccStorageBucketDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "google_storage_bucket" {
			continue
		}

		_, err := config.clientStorage.Buckets.Get(rs.Primary.ID).Do()
		if err == nil {
			return fmt.Errorf("Bucket still exists")
		}
	}

	return nil
}

func testAccStorageBucket_basic(bucketName string) string {
	return fmt.Sprintf(`
resource "google_storage_bucket" "bucket" {
	name = "%s"
}
`, bucketName)
}

func testAccStorageBucket_customAttributes(bucketName string) string {
	return fmt.Sprintf(`
resource "google_storage_bucket" "bucket" {
	name = "%s"
	predefined_acl = "publicReadWrite"
	location = "EU"
	force_destroy = "true"
}
`, bucketName)
}

func testAccStorageBucket_storageClass(bucketName, storageClass, location string) string {
	var locationBlock string
	if location != "" {
		locationBlock = fmt.Sprintf(`
	location = "%s"`, location)
	}
	return fmt.Sprintf(`
resource "google_storage_bucket" "bucket" {
	name = "%s"
	storage_class = "%s"%s
}
`, bucketName, storageClass, locationBlock)
}
