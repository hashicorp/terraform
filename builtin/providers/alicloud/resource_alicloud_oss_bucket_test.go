package alicloud

import (
	"fmt"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"log"
	"testing"
)

func TestAccAlicloudOssBucketBasic(t *testing.T) {
	var bucket oss.BucketInfo

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},

		// module name
		IDRefreshName: "alicloud_oss_bucket.basic",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckOssBucketDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(`
						resource "alicloud_oss_bucket" "basic" {
						bucket = "test-bucket-basic-%d"
						acl = "public-read"
						}
						`, acctest.RandInt()),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOssBucketExists(
						"alicloud_oss_bucket.basic", &bucket),
					resource.TestCheckResourceAttr(
						"alicloud_oss_bucket.basic",
						"location",
						"oss-cn-beijing"),
					resource.TestCheckResourceAttr(
						"alicloud_oss_bucket.basic",
						"acl",
						"public-read"),
				),
			},
		},
	})

}

func TestAccAlicloudOssBucketCors(t *testing.T) {
	var bucket oss.BucketInfo

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},

		// module name
		IDRefreshName: "alicloud_oss_bucket.cors",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckOssBucketDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(`
						resource "alicloud_oss_bucket" "cors" {
							bucket = "test-bucket-cors-%d"
							cors_rule ={
								allowed_origins=["*"]
								allowed_methods=["PUT","GET"]
								allowed_headers=["authorization"]
							}
							cors_rule ={
								allowed_origins=["http://www.a.com", "http://www.b.com"]
								allowed_methods=["GET"]
								allowed_headers=["authorization"]
								expose_headers=["x-oss-test","x-oss-test1"]
								max_age_seconds=100
							}
						}
						`, acctest.RandInt()),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOssBucketExists(
						"alicloud_oss_bucket.cors", &bucket),
					resource.TestCheckResourceAttr(
						"alicloud_oss_bucket.cors",
						"cors_rule.#",
						"2"),
					resource.TestCheckResourceAttr(
						"alicloud_oss_bucket.cors",
						"cors_rule.0.allowed_headers.0",
						"authorization"),
				),
			},
		},
	})
}

func TestAccAlicloudOssBucketWebsite(t *testing.T) {
	var bucket oss.BucketInfo

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},

		// module name
		IDRefreshName: "alicloud_oss_bucket.website",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckOssBucketDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(`
						resource "alicloud_oss_bucket" "website"{
							bucket = "test-bucket-website-%d"
							website = {
								index_document = "index.html"
								error_document = "error.html"
							}
						}
						`, acctest.RandInt()),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOssBucketExists(
						"alicloud_oss_bucket.website", &bucket),
					resource.TestCheckResourceAttr(
						"alicloud_oss_bucket.website",
						"website.#",
						"1"),
				),
			},
		},
	})
}
func TestAccAlicloudOssBucketLogging(t *testing.T) {
	var bucket oss.BucketInfo

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},

		// module name
		IDRefreshName: "alicloud_oss_bucket.logging",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckOssBucketDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(`
						resource "alicloud_oss_bucket" "target"{
							bucket = "test-target-%d"
						}
						resource "alicloud_oss_bucket" "logging" {
							bucket = "test-bucket-logging"
							logging {
								target_bucket = "${alicloud_oss_bucket.target.id}"
								target_prefix = "log/"
							}
						}
						`, acctest.RandInt()),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOssBucketExists(
						"alicloud_oss_bucket.target", &bucket),
					testAccCheckOssBucketExists(
						"alicloud_oss_bucket.logging", &bucket),
					resource.TestCheckResourceAttr(
						"alicloud_oss_bucket.logging",
						"logging.#",
						"1"),
				),
			},
		},
	})
}

func TestAccAlicloudOssBucketReferer(t *testing.T) {
	var bucket oss.BucketInfo

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},

		// module name
		IDRefreshName: "alicloud_oss_bucket.referer",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckOssBucketDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(`
						resource "alicloud_oss_bucket" "referer" {
							bucket = "test-bucket-referer-%d"
							referer_config {
								allow_empty = false
								referers = ["http://www.aliyun.com", "https://www.aliyun.com"]
							}
						}
						`, acctest.RandInt()),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOssBucketExists(
						"alicloud_oss_bucket.referer", &bucket),
					resource.TestCheckResourceAttr(
						"alicloud_oss_bucket.referer",
						"referer_config.#",
						"1"),
				),
			},
		},
	})
}
func TestAccAlicloudOssBucketLifecycle(t *testing.T) {
	var bucket oss.BucketInfo

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},

		// module name
		IDRefreshName: "alicloud_oss_bucket.lifecycle",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckOssBucketDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(`
						resource "alicloud_oss_bucket" "lifecycle"{
							bucket = "test-bucket-lifecycle-%d"
							lifecycle_rule {
								id = "rule1"
								prefix = "path1/"
								enabled = true
								expiration {
									days = 365
								}
							}
							lifecycle_rule {
								id = "rule2"
								prefix = "path2/"
								enabled = true
								expiration {
									date = "2018-01-12"
								}
							}
						}
						`, acctest.RandInt()),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOssBucketExists(
						"alicloud_oss_bucket.lifecycle", &bucket),
					resource.TestCheckResourceAttr(
						"alicloud_oss_bucket.lifecycle",
						"lifecycle_rule.#",
						"2"),
					resource.TestCheckResourceAttr(
						"alicloud_oss_bucket.lifecycle",
						"lifecycle_rule.0.id",
						"rule1"),
				),
			},
		},
	})
}
func testAccCheckOssBucketExists(n string, b *oss.BucketInfo) resource.TestCheckFunc {
	providers := []*schema.Provider{testAccProvider}
	return testAccCheckOssBucketExistsWithProviders(n, b, &providers)
}
func testAccCheckOssBucketExistsWithProviders(n string, b *oss.BucketInfo, providers *[]*schema.Provider) resource.TestCheckFunc {
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

			client := provider.Meta().(*AliyunClient)
			bucket, err := client.QueryOssBucketById(rs.Primary.ID)
			log.Printf("[WARN]get oss bucket %#v", bucket)
			if err == nil && bucket != nil {
				*b = *bucket
				return nil
			}

			// Verify the error is what we want
			e, _ := err.(*oss.ServiceError)
			if e.Code == OssBucketNotFound {
				continue
			}
			if err != nil {
				return err

			}
		}

		return fmt.Errorf("Bucket not found")
	}
}

func TestResourceAlicloudOssBucketAcl_validation(t *testing.T) {
	_, errors := validateOssBucketAcl("incorrect", "acl")
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
		_, errors := validateOssBucketAcl(tc.Value, "acl")
		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected not to trigger a validation error")
		}
	}
}

func testAccCheckOssBucketDestroy(s *terraform.State) error {
	return testAccCheckOssBucketDestroyWithProvider(s, testAccProvider)
}

func testAccCheckOssBucketDestroyWithProvider(s *terraform.State, provider *schema.Provider) error {
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
