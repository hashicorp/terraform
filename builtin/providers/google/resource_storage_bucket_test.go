package google

import (
	"bytes"
	"fmt"
	"log"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

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

func TestAccStorageBucket_cors(t *testing.T) {
	var bucket storage.Bucket
	bucketName := fmt.Sprintf("tf-test-acl-bucket-%d", acctest.RandInt())

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccStorageBucketDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testGoogleStorageBucketsCors(bucketName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckStorageBucketExists(
						"google_storage_bucket.bucket", bucketName, &bucket),
				),
			},
		},
	})

	if len(bucket.Cors) != 2 {
		t.Errorf("Expected # of cors elements to be 2, got %d", len(bucket.Cors))
	}

	firstArr := bucket.Cors[0]
	if firstArr.MaxAgeSeconds != 10 {
		t.Errorf("Expected first block's MaxAgeSeconds to be 10, got %d", firstArr.MaxAgeSeconds)
	}

	for i, v := range []string{"abc", "def"} {
		if firstArr.Origin[i] != v {
			t.Errorf("Expected value in first block origin to be to be %v, got %v", v, firstArr.Origin[i])
		}
	}

	for i, v := range []string{"a1a"} {
		if firstArr.Method[i] != v {
			t.Errorf("Expected value in first block method to be to be %v, got %v", v, firstArr.Method[i])
		}
	}

	for i, v := range []string{"123", "456", "789"} {
		if firstArr.ResponseHeader[i] != v {
			t.Errorf("Expected value in first block response headerto be to be %v, got %v", v, firstArr.ResponseHeader[i])
		}
	}

	secondArr := bucket.Cors[1]
	if secondArr.MaxAgeSeconds != 5 {
		t.Errorf("Expected second block's MaxAgeSeconds to be 5, got %d", secondArr.MaxAgeSeconds)
	}

	for i, v := range []string{"ghi", "jkl"} {
		if secondArr.Origin[i] != v {
			t.Errorf("Expected value in second block origin to be to be %v, got %v", v, secondArr.Origin[i])
		}
	}

	for i, v := range []string{"z9z"} {
		if secondArr.Method[i] != v {
			t.Errorf("Expected value in second block method to be to be %v, got %v", v, secondArr.Method[i])
		}
	}

	for i, v := range []string{"000"} {
		if secondArr.ResponseHeader[i] != v {
			t.Errorf("Expected value in second block response headerto be to be %v, got %v", v, secondArr.ResponseHeader[i])
		}
	}
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

func testGoogleStorageBucketsCors(bucketName string) string {
	return fmt.Sprintf(`
resource "google_storage_bucket" "bucket" {
	name = "%s"
	cors {
	  origin = ["abc", "def"]
	  method = ["a1a"]
	  response_header = ["123", "456", "789"]
	  max_age_seconds = 10
	}

	cors {
	  origin = ["ghi", "jkl"]
	  method = ["z9z"]
	  response_header = ["000"]
	  max_age_seconds = 5
	}
}
`, bucketName)
}
