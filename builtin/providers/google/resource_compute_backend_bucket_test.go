package google

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"google.golang.org/api/compute/v1"
)

func TestAccComputeBackendBucket_basic(t *testing.T) {
	backendName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	storageName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	var svc compute.BackendBucket

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeBackendBucketDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeBackendBucket_basic(backendName, storageName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeBackendBucketExists(
						"google_compute_backend_bucket.foobar", &svc),
				),
			},
		},
	})

	if svc.BucketName != storageName {
		t.Errorf("Expected BucketName to be %q, got %q", storageName, svc.BucketName)
	}
}

func TestAccComputeBackendBucket_basicModified(t *testing.T) {
	backendName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	storageName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	secondStorageName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	var svc compute.BackendBucket

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeBackendBucketDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeBackendBucket_basic(backendName, storageName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeBackendBucketExists(
						"google_compute_backend_bucket.foobar", &svc),
				),
			},
			resource.TestStep{
				Config: testAccComputeBackendBucket_basicModified(
					backendName, storageName, secondStorageName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeBackendBucketExists(
						"google_compute_backend_bucket.foobar", &svc),
				),
			},
		},
	})

	if svc.BucketName != secondStorageName {
		t.Errorf("Expected BucketName to be %q, got %q", secondStorageName, svc.BucketName)
	}
}

func testAccCheckComputeBackendBucketDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "google_compute_backend_bucket" {
			continue
		}

		_, err := config.clientCompute.BackendBuckets.Get(
			config.Project, rs.Primary.ID).Do()
		if err == nil {
			return fmt.Errorf("Backend bucket %s still exists", rs.Primary.ID)
		}
	}

	return nil
}

func testAccCheckComputeBackendBucketExists(n string, svc *compute.BackendBucket) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)

		found, err := config.clientCompute.BackendBuckets.Get(
			config.Project, rs.Primary.ID).Do()
		if err != nil {
			return err
		}

		if found.Name != rs.Primary.ID {
			return fmt.Errorf("Backend bucket %s not found", rs.Primary.ID)
		}

		*svc = *found

		return nil
	}
}

func TestAccComputeBackendBucket_withCdnEnabled(t *testing.T) {
	backendName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	storageName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	var svc compute.BackendBucket

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeBackendBucketDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeBackendBucket_withCdnEnabled(
					backendName, storageName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeBackendBucketExists(
						"google_compute_backend_bucket.foobar", &svc),
				),
			},
		},
	})

	if svc.EnableCdn != true {
		t.Errorf("Expected EnableCdn == true, got %t", svc.EnableCdn)
	}
}

func testAccComputeBackendBucket_basic(backendName, storageName string) string {
	return fmt.Sprintf(`
resource "google_compute_backend_bucket" "foobar" {
  name        = "%s"
  bucket_name = "${google_storage_bucket.bucket_one.name}"
}

resource "google_storage_bucket" "bucket_one" {
  name     = "%s"
  location = "EU"
}
`, backendName, storageName)
}

func testAccComputeBackendBucket_basicModified(backendName, bucketOne, bucketTwo string) string {
	return fmt.Sprintf(`
resource "google_compute_backend_bucket" "foobar" {
  name        = "%s"
  bucket_name = "${google_storage_bucket.bucket_two.name}"
}

resource "google_storage_bucket" "bucket_one" {
  name     = "%s"
  location = "EU"
}

resource "google_storage_bucket" "bucket_two" {
  name     = "%s"
  location = "EU"
}
`, backendName, bucketOne, bucketTwo)
}

func testAccComputeBackendBucket_withCdnEnabled(backendName, storageName string) string {
	return fmt.Sprintf(`
resource "google_compute_backend_bucket" "foobar" {
  name        = "%s"
  bucket_name = "${google_storage_bucket.bucket.name}"
  enable_cdn  = true
}

resource "google_storage_bucket" "bucket" {
  name     = "%s"
  location = "EU"
}
`, backendName, storageName)
}
