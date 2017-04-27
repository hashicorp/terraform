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

func TestAccFastlyServiceV1_healthcheck_basic(t *testing.T) {
	var service gofastly.ServiceDetail
	name := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	domainName := fmt.Sprintf("%s.notadomain.com", acctest.RandString(10))

	log1 := gofastly.HealthCheck{
		Version:          1,
		Name:             "example-healthcheck1",
		Host:             "example1.com",
		Path:             "/test1.txt",
		CheckInterval:    4000,
		ExpectedResponse: 200,
		HTTPVersion:      "1.1",
		Initial:          2,
		Method:           "HEAD",
		Threshold:        3,
		Timeout:          5000,
		Window:           5,
	}

	log2 := gofastly.HealthCheck{
		Version:          1,
		Name:             "example-healthcheck2",
		Host:             "example2.com",
		Path:             "/test2.txt",
		CheckInterval:    4500,
		ExpectedResponse: 404,
		HTTPVersion:      "1.0",
		Initial:          1,
		Method:           "POST",
		Threshold:        4,
		Timeout:          4000,
		Window:           10,
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckServiceV1Destroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccServiceV1HealthCheckConfig(name, domainName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckServiceV1Exists("fastly_service_v1.foo", &service),
					testAccCheckFastlyServiceV1HealthCheckAttributes(&service, []*gofastly.HealthCheck{&log1}),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "name", name),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "healthcheck.#", "1"),
				),
			},

			resource.TestStep{
				Config: testAccServiceV1HealthCheckConfig_update(name, domainName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckServiceV1Exists("fastly_service_v1.foo", &service),
					testAccCheckFastlyServiceV1HealthCheckAttributes(&service, []*gofastly.HealthCheck{&log1, &log2}),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "name", name),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "healthcheck.#", "2"),
				),
			},
		},
	})
}

func testAccCheckFastlyServiceV1HealthCheckAttributes(service *gofastly.ServiceDetail, healthchecks []*gofastly.HealthCheck) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		conn := testAccProvider.Meta().(*FastlyClient).conn
		healthcheckList, err := conn.ListHealthChecks(&gofastly.ListHealthChecksInput{
			Service: service.ID,
			Version: service.ActiveVersion.Number,
		})

		if err != nil {
			return fmt.Errorf("[ERR] Error looking up Healthcheck for (%s), version (%v): %s", service.Name, service.ActiveVersion.Number, err)
		}

		if len(healthcheckList) != len(healthchecks) {
			return fmt.Errorf("Healthcheck List count mismatch, expected (%d), got (%d)", len(healthchecks), len(healthcheckList))
		}

		var found int
		for _, h := range healthchecks {
			for _, lh := range healthcheckList {
				if h.Name == lh.Name {
					// we don't know these things ahead of time, so populate them now
					h.ServiceID = service.ID
					h.Version = service.ActiveVersion.Number
					if !reflect.DeepEqual(h, lh) {
						return fmt.Errorf("Bad match Healthcheck match, expected (%#v), got (%#v)", h, lh)
					}
					found++
				}
			}
		}

		if found != len(healthchecks) {
			return fmt.Errorf("Error matching Healthcheck rules")
		}

		return nil
	}
}

func testAccServiceV1HealthCheckConfig(name, domain string) string {
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

  healthcheck {
		name              = "example-healthcheck1"
		host              = "example1.com"
		path              = "/test1.txt"
		check_interval    = 4000
		expected_response = 200
		http_version      = "1.1"
		initial           = 2
		method            = "HEAD"
		threshold         = 3
		timeout           = 5000
		window            = 5
  }

  force_destroy = true
}`, name, domain)
}

func testAccServiceV1HealthCheckConfig_update(name, domain string) string {
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

	healthcheck {
		name              = "example-healthcheck1"
		host              = "example1.com"
		path              = "/test1.txt"
		check_interval    = 4000
		expected_response = 200
		http_version      = "1.1"
		initial           = 2
		method            = "HEAD"
		threshold         = 3
		timeout           = 5000
		window            = 5
  }

	healthcheck {
		name              = "example-healthcheck2"
		host              = "example2.com"
		path              = "/test2.txt"
		check_interval    = 4500
		expected_response = 404
		http_version      = "1.0"
		initial           = 1
		method            = "POST"
		threshold         = 4
		timeout           = 4000
		window            = 10
  }

  force_destroy = true
}`, name, domain)
}
