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

func TestAccFastlyServiceV1_response_object_basic(t *testing.T) {
	var service gofastly.ServiceDetail
	name := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	domainName1 := fmt.Sprintf("%s.notadomain.com", acctest.RandString(10))

	log1 := gofastly.ResponseObject{
		Version:          1,
		Name:             "responseObjecttesting",
		Status:           200,
		Response:         "OK",
		Content:          "test content",
		ContentType:      "text/html",
		RequestCondition: "test-request-condition",
		CacheCondition:   "test-cache-condition",
	}

	log2 := gofastly.ResponseObject{
		Version:          1,
		Name:             "responseObjecttesting2",
		Status:           404,
		Response:         "Not Found",
		Content:          "some, other, content",
		ContentType:      "text/csv",
		RequestCondition: "another-test-request-condition",
		CacheCondition:   "another-test-cache-condition",
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckServiceV1Destroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccServiceV1ResponseObjectConfig(name, domainName1),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckServiceV1Exists("fastly_service_v1.foo", &service),
					testAccCheckFastlyServiceV1ResponseObjectAttributes(&service, []*gofastly.ResponseObject{&log1}),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "name", name),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "response_object.#", "1"),
				),
			},

			resource.TestStep{
				Config: testAccServiceV1ResponseObjectConfig_update(name, domainName1),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckServiceV1Exists("fastly_service_v1.foo", &service),
					testAccCheckFastlyServiceV1ResponseObjectAttributes(&service, []*gofastly.ResponseObject{&log1, &log2}),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "name", name),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "response_object.#", "2"),
				),
			},
		},
	})
}

func testAccCheckFastlyServiceV1ResponseObjectAttributes(service *gofastly.ServiceDetail, responseObjects []*gofastly.ResponseObject) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		conn := testAccProvider.Meta().(*FastlyClient).conn
		responseObjectList, err := conn.ListResponseObjects(&gofastly.ListResponseObjectsInput{
			Service: service.ID,
			Version: service.ActiveVersion.Number,
		})

		if err != nil {
			return fmt.Errorf("[ERR] Error looking up Response Object for (%s), version (%v): %s", service.Name, service.ActiveVersion.Number, err)
		}

		if len(responseObjectList) != len(responseObjects) {
			return fmt.Errorf("Response Object List count mismatch, expected (%d), got (%d)", len(responseObjects), len(responseObjectList))
		}

		var found int
		for _, p := range responseObjects {
			for _, lp := range responseObjectList {
				if p.Name == lp.Name {
					// we don't know these things ahead of time, so populate them now
					p.ServiceID = service.ID
					p.Version = service.ActiveVersion.Number
					if !reflect.DeepEqual(p, lp) {
						return fmt.Errorf("Bad match Response Object match, expected (%#v), got (%#v)", p, lp)
					}
					found++
				}
			}
		}

		if found != len(responseObjects) {
			return fmt.Errorf("Error matching Response Object rules")
		}

		return nil
	}
}

func testAccServiceV1ResponseObjectConfig(name, domain string) string {
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
    name      = "test-request-condition"
    type      = "REQUEST"
    priority  = 5
    statement = "req.url ~ \"^/foo/bar$\""
  }

	condition {
    name      = "test-cache-condition"
    type      = "CACHE"
    priority  = 9
    statement = "req.url ~ \"^/articles/\""
  }

  response_object {
		name              = "responseObjecttesting"
		status            = 200
		response          = "OK"
		content           = "test content"
		content_type      = "text/html"
		request_condition = "test-request-condition"
		cache_condition   = "test-cache-condition"
  }

  force_destroy = true
}`, name, domain)
}

func testAccServiceV1ResponseObjectConfig_update(name, domain string) string {
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
    name      = "test-cache-condition"
    type      = "CACHE"
    priority  = 9
    statement = "req.url ~ \"^/articles/\""
  }

	condition {
    name      = "another-test-cache-condition"
    type      = "CACHE"
    priority  = 7
    statement = "req.url ~ \"^/stories/\""
  }

	condition {
    name      = "test-request-condition"
    type      = "REQUEST"
    priority  = 5
    statement = "req.url ~ \"^/foo/bar$\""
  }

	condition {
    name      = "another-test-request-condition"
    type      = "REQUEST"
    priority  = 10
    statement = "req.url ~ \"^/articles$\""
  }

  response_object {
		name              = "responseObjecttesting"
		status            = 200
		response          = "OK"
		content           = "test content"
		content_type      = "text/html"
		request_condition = "test-request-condition"
		cache_condition   = "test-cache-condition"
  }

  response_object {
		name              = "responseObjecttesting2"
		status            = 404
		response          = "Not Found"
		content           = "some, other, content"
		content_type      = "text/csv"
		request_condition = "another-test-request-condition"
		cache_condition   = "another-test-cache-condition"
  }

  force_destroy = true
}`, name, domain)
}
