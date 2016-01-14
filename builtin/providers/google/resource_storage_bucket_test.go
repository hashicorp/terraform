package google

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"google.golang.org/api/googleapi"
	storage "google.golang.org/api/storage/v1"
)

func TestAccStorage_basic(t *testing.T) {
	bucketName := fmt.Sprintf("tf-test-acl-bucket-%d", acctest.RandInt())

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccGoogleStorageDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testGoogleStorageBucketsReaderDefaults(bucketName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudStorageBucketExists(
						"google_storage_bucket.bucket", bucketName),
					resource.TestCheckResourceAttr(
						"google_storage_bucket.bucket", "location", "US"),
					resource.TestCheckResourceAttr(
						"google_storage_bucket.bucket", "force_destroy", "false"),
				),
			},
		},
	})
}

func TestAccStorageCustomAttributes(t *testing.T) {
	bucketName := fmt.Sprintf("tf-test-acl-bucket-%d", acctest.RandInt())

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccGoogleStorageDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testGoogleStorageBucketsReaderCustomAttributes(bucketName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudStorageBucketExists(
						"google_storage_bucket.bucket", bucketName),
					resource.TestCheckResourceAttr(
						"google_storage_bucket.bucket", "location", "EU"),
					resource.TestCheckResourceAttr(
						"google_storage_bucket.bucket", "force_destroy", "true"),
				),
			},
		},
	})
}

func TestAccStorageBucketUpdate(t *testing.T) {
	bucketName := fmt.Sprintf("tf-test-acl-bucket-%d", acctest.RandInt())

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccGoogleStorageDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testGoogleStorageBucketsReaderDefaults(bucketName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudStorageBucketExists(
						"google_storage_bucket.bucket", bucketName),
					resource.TestCheckResourceAttr(
						"google_storage_bucket.bucket", "location", "US"),
					resource.TestCheckResourceAttr(
						"google_storage_bucket.bucket", "force_destroy", "false"),
				),
			},
			resource.TestStep{
				Config: testGoogleStorageBucketsReaderCustomAttributes(bucketName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudStorageBucketExists(
						"google_storage_bucket.bucket", bucketName),
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

func TestAccStorageForceDestroy(t *testing.T) {
	bucketName := fmt.Sprintf("tf-test-acl-bucket-%d", acctest.RandInt())

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccGoogleStorageDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testGoogleStorageBucketsReaderCustomAttributes(bucketName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudStorageBucketExists(
						"google_storage_bucket.bucket", bucketName),
				),
			},
			resource.TestStep{
				Config: testGoogleStorageBucketsReaderCustomAttributes(bucketName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudStorageBucketPutItem(bucketName),
				),
			},
			resource.TestStep{
				Config: "",
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudStorageBucketMissing(bucketName),
				),
			},
		},
	})
}

func testAccCheckCloudStorageBucketExists(n string, bucketName string) resource.TestCheckFunc {
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
		return nil
	}
}

func testAccCheckCloudStorageBucketPutItem(bucketName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		config := testAccProvider.Meta().(*Config)

		data := bytes.NewBufferString("test")
		dataReader := bytes.NewReader(data.Bytes())
		object := &storage.Object{Name: "bucketDestroyTestFile"}

		// This needs to use Media(io.Reader) call, otherwise it does not go to /upload API and fails
		if res, err := config.clientStorage.Objects.Insert(bucketName, object).Media(dataReader).Do(); err == nil {
			fmt.Printf("Created object %v at location %v\n\n", res.Name, res.SelfLink)
		} else {
			return fmt.Errorf("Objects.Insert failed: %v", err)
		}

		return nil
	}
}

func testAccCheckCloudStorageBucketMissing(bucketName string) resource.TestCheckFunc {
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

func testAccGoogleStorageDestroy(s *terraform.State) error {
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

func testGoogleStorageBucketsReaderDefaults(bucketName string) string {
	return fmt.Sprintf(`
resource "google_storage_bucket" "bucket" {
	name = "%s"
}
`, bucketName)
}

func testGoogleStorageBucketsReaderCustomAttributes(bucketName string) string {
	return fmt.Sprintf(`
resource "google_storage_bucket" "bucket" {
	name = "%s"
	predefined_acl = "publicReadWrite"
	location = "EU"
	force_destroy = "true"
}
`, bucketName)
}
