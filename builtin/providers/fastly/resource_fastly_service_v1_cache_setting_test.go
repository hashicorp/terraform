package fastly

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	gofastly "github.com/sethvargo/go-fastly"
)

func TestAccFastlyServiceV1CacheSetting_basic(t *testing.T) {
	var service gofastly.ServiceDetail
	name := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	domainName1 := fmt.Sprintf("%s.notadomain.com", acctest.RandString(10))

	cq1 := gofastly.CacheSetting{
		Name:           "alt_backend",
		Action:         "pass",
		StaleTTL:       uint(3600),
		CacheCondition: "serve_alt_backend",
	}

	cq2 := gofastly.CacheSetting{
		Name:           "cache_backend",
		Action:         "restart",
		StaleTTL:       uint(1600),
		CacheCondition: "cache_alt_backend",
		TTL:            uint(300),
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckServiceV1Destroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccServiceV1CacheSetting(name, domainName1),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckServiceV1Exists("fastly_service_v1.foo", &service),
					testAccCheckFastlyServiceV1CacheSettingsAttributes(&service, []*gofastly.CacheSetting{&cq1}),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "name", name),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "cache_setting.#", "1"),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "condition.#", "1"),
				),
			},

			resource.TestStep{
				Config: testAccServiceV1CacheSetting_update(name, domainName1),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckServiceV1Exists("fastly_service_v1.foo", &service),
					testAccCheckFastlyServiceV1CacheSettingsAttributes(&service, []*gofastly.CacheSetting{&cq1, &cq2}),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "cache_setting.#", "2"),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "condition.#", "2"),
				),
			},
		},
	})
}

func testAccCheckFastlyServiceV1CacheSettingsAttributes(service *gofastly.ServiceDetail, rqs []*gofastly.CacheSetting) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		conn := testAccProvider.Meta().(*FastlyClient).conn
		rqList, err := conn.ListCacheSettings(&gofastly.ListCacheSettingsInput{
			Service: service.ID,
			Version: service.ActiveVersion.Number,
		})

		if err != nil {
			return fmt.Errorf("[ERR] Error looking up Request Setting for (%s), version (%v): %s", service.Name, service.ActiveVersion.Number, err)
		}

		if len(rqList) != len(rqs) {
			return fmt.Errorf("Request Setting List count mismatch, expected (%d), got (%d)", len(rqs), len(rqList))
		}

		var found int
		for _, r := range rqs {
			for _, lr := range rqList {
				if r.Name == lr.Name {
					// we don't know these things ahead of time, so populate them now
					r.ServiceID = service.ID
					r.Version = service.ActiveVersion.Number
					if !reflect.DeepEqual(r, lr) {
						return fmt.Errorf("Bad match Request Setting match, expected (%#v), got (%#v)", r, lr)
					}
					found++
				}
			}
		}

		if found != len(rqs) {
			return fmt.Errorf("Error matching Request Setting rules (%d/%d)", found, len(rqs))
		}

		return nil
	}
}

func testAccServiceV1CacheSetting(name, domain string) string {
	return fmt.Sprintf(`
resource "fastly_service_v1" "foo" {
  name = "%s"

  domain {
    name    = "%s"
    comment = "demo"
  }

  backend {
    address = "tftesting.tftesting.net.s3-website-us-west-2.amazonaws.com"
    name    = "AWS S3 hosting"
    port    = 80
  }

  backend {
    address = "tftestingother.tftesting.net.s3-website-us-west-2.amazonaws.com"
    name    = "OtherAWSS3hosting"
    port    = 80
  }

  condition {
    name      = "serve_alt_backend"
    type      = "CACHE"
    priority  = 10
    statement = "req.url ~ \"^/alt/\""
  }

  cache_setting {
    name            = "alt_backend"
    stale_ttl       = 3600
    cache_condition = "serve_alt_backend"
    action          = "pass"
  }

  default_host = "tftesting.tftesting.net.s3-website-us-west-2.amazonaws.com"

  force_destroy = true
}`, name, domain)
}

func testAccServiceV1CacheSetting_update(name, domain string) string {
	return fmt.Sprintf(`
resource "fastly_service_v1" "foo" {
  name = "%s"

  domain {
    name    = "%s"
    comment = "demo"
  }

  backend {
    address = "tftesting.tftesting.net.s3-website-us-west-2.amazonaws.com"
    name    = "AWS S3 hosting"
    port    = 80
  }

  backend {
    address = "tftestingother.tftesting.net.s3-website-us-west-2.amazonaws.com"
    name    = "OtherAWSS3hosting"
    port    = 80
  }

  condition {
    name      = "serve_alt_backend"
    type      = "CACHE"
    priority  = 10
    statement = "req.url ~ \"^/alt/\""
  }

  condition {
    name      = "cache_alt_backend"
    type      = "CACHE"
    priority  = 20
    statement = "req.url ~ \"^/cache/\""
  }

  cache_setting {
    name            = "alt_backend"
    stale_ttl       = 3600
    cache_condition = "serve_alt_backend"
    action          = "pass"
  }

  cache_setting {
    name            = "cache_backend"
    stale_ttl       = 1600
    cache_condition = "cache_alt_backend"
    action          = "restart"
    ttl             = 300
  }

  default_host = "tftesting.tftesting.net.s3-website-us-west-2.amazonaws.com"

  force_destroy = true
}`, name, domain)
}
