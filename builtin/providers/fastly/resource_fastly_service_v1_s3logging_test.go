package fastly

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	gofastly "github.com/sethvargo/go-fastly"
)

func TestAccFastlyServiceV1_s3logging_basic(t *testing.T) {
	var service gofastly.ServiceDetail
	name := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	domainName1 := fmt.Sprintf("%s.notadomain.com", acctest.RandString(10))

	log1 := gofastly.S3{
		Version:           1,
		Name:              "somebucketlog",
		BucketName:        "fastlytestlogging",
		Domain:            "s3-us-west-2.amazonaws.com",
		AccessKey:         "somekey",
		SecretKey:         "somesecret",
		Period:            uint(3600),
		GzipLevel:         uint(0),
		Format:            "%h %l %u %t %r %>s",
		FormatVersion:     1,
		TimestampFormat:   "%Y-%m-%dT%H:%M:%S.000",
		ResponseCondition: "response_condition_test",
	}

	log2 := gofastly.S3{
		Version:         1,
		Name:            "someotherbucketlog",
		BucketName:      "fastlytestlogging2",
		Domain:          "s3-us-west-2.amazonaws.com",
		AccessKey:       "someotherkey",
		SecretKey:       "someothersecret",
		GzipLevel:       uint(3),
		Period:          uint(60),
		Format:          "%h %l %u %t %r %>s",
		FormatVersion:   1,
		TimestampFormat: "%Y-%m-%dT%H:%M:%S.000",
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckServiceV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccServiceV1S3LoggingConfig(name, domainName1),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckServiceV1Exists("fastly_service_v1.foo", &service),
					testAccCheckFastlyServiceV1S3LoggingAttributes(&service, []*gofastly.S3{&log1}),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "name", name),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "s3logging.#", "1"),
				),
			},

			{
				Config: testAccServiceV1S3LoggingConfig_update(name, domainName1),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckServiceV1Exists("fastly_service_v1.foo", &service),
					testAccCheckFastlyServiceV1S3LoggingAttributes(&service, []*gofastly.S3{&log1, &log2}),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "name", name),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "s3logging.#", "2"),
				),
			},
		},
	})
}

// Tests that s3_access_key and s3_secret_key are read from the env
func TestAccFastlyServiceV1_s3logging_s3_env(t *testing.T) {
	var service gofastly.ServiceDetail
	name := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	domainName1 := fmt.Sprintf("%s.notadomain.com", acctest.RandString(10))

	// set env Vars to something we expect
	resetEnv := setEnv("someEnv", t)
	defer resetEnv()

	log3 := gofastly.S3{
		Version:         1,
		Name:            "somebucketlog",
		BucketName:      "fastlytestlogging",
		Domain:          "s3-us-west-2.amazonaws.com",
		AccessKey:       "someEnv",
		SecretKey:       "someEnv",
		Period:          uint(3600),
		GzipLevel:       uint(0),
		Format:          "%h %l %u %t %r %>s",
		FormatVersion:   1,
		TimestampFormat: "%Y-%m-%dT%H:%M:%S.000",
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckServiceV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccServiceV1S3LoggingConfig_env(name, domainName1),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckServiceV1Exists("fastly_service_v1.foo", &service),
					testAccCheckFastlyServiceV1S3LoggingAttributes(&service, []*gofastly.S3{&log3}),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "name", name),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "s3logging.#", "1"),
				),
			},
		},
	})
}

func TestAccFastlyServiceV1_s3logging_formatVersion(t *testing.T) {
	var service gofastly.ServiceDetail
	name := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	domainName1 := fmt.Sprintf("%s.notadomain.com", acctest.RandString(10))

	log1 := gofastly.S3{
		Version:         1,
		Name:            "somebucketlog",
		BucketName:      "fastlytestlogging",
		Domain:          "s3-us-west-2.amazonaws.com",
		AccessKey:       "somekey",
		SecretKey:       "somesecret",
		Period:          uint(3600),
		GzipLevel:       uint(0),
		Format:          "%a %l %u %t %m %U%q %H %>s %b %T",
		FormatVersion:   2,
		TimestampFormat: "%Y-%m-%dT%H:%M:%S.000",
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckServiceV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccServiceV1S3LoggingConfig_formatVersion(name, domainName1),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckServiceV1Exists("fastly_service_v1.foo", &service),
					testAccCheckFastlyServiceV1S3LoggingAttributes(&service, []*gofastly.S3{&log1}),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "name", name),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "s3logging.#", "1"),
				),
			},
		},
	})
}

func testAccCheckFastlyServiceV1S3LoggingAttributes(service *gofastly.ServiceDetail, s3s []*gofastly.S3) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		conn := testAccProvider.Meta().(*FastlyClient).conn
		s3List, err := conn.ListS3s(&gofastly.ListS3sInput{
			Service: service.ID,
			Version: service.ActiveVersion.Number,
		})

		if err != nil {
			return fmt.Errorf("[ERR] Error looking up S3 Logging for (%s), version (%v): %s", service.Name, service.ActiveVersion.Number, err)
		}

		if len(s3List) != len(s3s) {
			return fmt.Errorf("S3 List count mismatch, expected (%d), got (%d)", len(s3s), len(s3List))
		}

		var found int
		for _, s := range s3s {
			for _, ls := range s3List {
				if s.Name == ls.Name {
					// we don't know these things ahead of time, so populate them now
					s.ServiceID = service.ID
					s.Version = service.ActiveVersion.Number
					// We don't track these, so clear them out because we also wont know
					// these ahead of time
					ls.CreatedAt = nil
					ls.UpdatedAt = nil
					if !reflect.DeepEqual(s, ls) {
						return fmt.Errorf("Bad match S3 logging match, expected (%#v), got (%#v)", s, ls)
					}
					found++
				}
			}
		}

		if found != len(s3s) {
			return fmt.Errorf("Error matching S3 Logging rules")
		}

		return nil
	}
}

func testAccServiceV1S3LoggingConfig(name, domain string) string {
	return fmt.Sprintf(`
resource "fastly_service_v1" "foo" {
  name = "%s"

  domain {
    name    = "%s"
    comment = "tf-testing-domain"
  }

  backend {
    address = "aws.amazon.com"
    name    = "amazon docs"
  }

	condition {
    name      = "response_condition_test"
    type      = "RESPONSE"
    priority  = 8
    statement = "resp.status == 418"
  }

  s3logging {
    name               = "somebucketlog"
    bucket_name        = "fastlytestlogging"
    domain             = "s3-us-west-2.amazonaws.com"
    s3_access_key      = "somekey"
    s3_secret_key      = "somesecret"
		response_condition = "response_condition_test"
  }

  force_destroy = true
}`, name, domain)
}

func testAccServiceV1S3LoggingConfig_update(name, domain string) string {
	return fmt.Sprintf(`
resource "fastly_service_v1" "foo" {
  name = "%s"

  domain {
    name    = "%s"
    comment = "tf-testing-domain"
  }

  backend {
    address = "aws.amazon.com"
    name    = "amazon docs"
  }

	condition {
    name      = "response_condition_test"
    type      = "RESPONSE"
    priority  = 8
    statement = "resp.status == 418"
  }

  s3logging {
    name               = "somebucketlog"
    bucket_name        = "fastlytestlogging"
    domain             = "s3-us-west-2.amazonaws.com"
    s3_access_key      = "somekey"
    s3_secret_key      = "somesecret"
		response_condition = "response_condition_test"
  }

  s3logging {
    name          = "someotherbucketlog"
    bucket_name   = "fastlytestlogging2"
    domain        = "s3-us-west-2.amazonaws.com"
    s3_access_key = "someotherkey"
    s3_secret_key = "someothersecret"
    period        = 60
    gzip_level    = 3
  }

  force_destroy = true
}`, name, domain)
}

func testAccServiceV1S3LoggingConfig_env(name, domain string) string {
	return fmt.Sprintf(`
resource "fastly_service_v1" "foo" {
  name = "%s"

  domain {
    name    = "%s"
    comment = "tf-testing-domain"
  }

  backend {
    address = "aws.amazon.com"
    name    = "amazon docs"
  }

  s3logging {
    name          = "somebucketlog"
    bucket_name   = "fastlytestlogging"
    domain        = "s3-us-west-2.amazonaws.com"
  }

  force_destroy = true
}`, name, domain)
}

func testAccServiceV1S3LoggingConfig_formatVersion(name, domain string) string {
	return fmt.Sprintf(`
resource "fastly_service_v1" "foo" {
  name = "%s"

  domain {
    name    = "%s"
    comment = "tf-testing-domain"
  }

  backend {
    address = "aws.amazon.com"
    name    = "amazon docs"
  }

  s3logging {
    name           = "somebucketlog"
    bucket_name    = "fastlytestlogging"
    domain         = "s3-us-west-2.amazonaws.com"
    s3_access_key  = "somekey"
    s3_secret_key  = "somesecret"
    format         = "%%a %%l %%u %%t %%m %%U%%q %%H %%>s %%b %%T"
    format_version = 2
  }

  force_destroy = true
}`, name, domain)
}

func setEnv(s string, t *testing.T) func() {
	e := getEnv()
	// Set all the envs to a dummy value
	if err := os.Setenv("FASTLY_S3_ACCESS_KEY", s); err != nil {
		t.Fatalf("Error setting env var AWS_ACCESS_KEY_ID: %s", err)
	}
	if err := os.Setenv("FASTLY_S3_SECRET_KEY", s); err != nil {
		t.Fatalf("Error setting env var FASTLY_S3_SECRET_KEY: %s", err)
	}

	return func() {
		// re-set all the envs we unset above
		if err := os.Setenv("FASTLY_S3_ACCESS_KEY", e.Key); err != nil {
			t.Fatalf("Error resetting env var AWS_ACCESS_KEY_ID: %s", err)
		}
		if err := os.Setenv("FASTLY_S3_SECRET_KEY", e.Secret); err != nil {
			t.Fatalf("Error resetting env var FASTLY_S3_SECRET_KEY: %s", err)
		}
	}
}

// struct to preserve the current environment
type currentEnv struct {
	Key, Secret string
}

func getEnv() *currentEnv {
	// Grab any existing Fastly AWS S3 keys and preserve, in the off chance
	// they're actually set in the enviornment
	return &currentEnv{
		Key:    os.Getenv("FASTLY_S3_ACCESS_KEY"),
		Secret: os.Getenv("FASTLY_S3_SECRET_KEY"),
	}
}
