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

func TestFastlyServiceV1_BuildHeaders(t *testing.T) {
	cases := []struct {
		remote *gofastly.CreateHeaderInput
		local  map[string]interface{}
	}{
		{
			remote: &gofastly.CreateHeaderInput{
				Name:        "someheadder",
				Action:      gofastly.HeaderActionDelete,
				IgnoreIfSet: gofastly.CBool(true),
				Type:        gofastly.HeaderTypeCache,
				Destination: "http.aws-id",
				Priority:    uint(100),
			},
			local: map[string]interface{}{
				"name":               "someheadder",
				"action":             "delete",
				"ignore_if_set":      true,
				"destination":        "http.aws-id",
				"priority":           100,
				"source":             "",
				"regex":              "",
				"substitution":       "",
				"request_condition":  "",
				"cache_condition":    "",
				"response_condition": "",
				"type":               "cache",
			},
		},
		{
			remote: &gofastly.CreateHeaderInput{
				Name:        "someheadder",
				Action:      gofastly.HeaderActionSet,
				IgnoreIfSet: gofastly.CBool(false),
				Type:        gofastly.HeaderTypeCache,
				Destination: "http.aws-id",
				Priority:    uint(100),
				Source:      "http.server-name",
			},
			local: map[string]interface{}{
				"name":               "someheadder",
				"action":             "set",
				"ignore_if_set":      false,
				"destination":        "http.aws-id",
				"priority":           100,
				"source":             "http.server-name",
				"regex":              "",
				"substitution":       "",
				"request_condition":  "",
				"cache_condition":    "",
				"response_condition": "",
				"type":               "cache",
			},
		},
	}

	for _, c := range cases {
		out, _ := buildHeader(c.local)
		if !reflect.DeepEqual(out, c.remote) {
			t.Fatalf("Error matching:\nexpected: %#v\ngot: %#v", c.remote, out)
		}
	}
}

func TestAccFastlyServiceV1_headers_basic(t *testing.T) {
	var service gofastly.ServiceDetail
	name := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	domainName1 := fmt.Sprintf("%s.notadomain.com", acctest.RandString(10))

	log1 := gofastly.Header{
		Version:     1,
		Name:        "remove x-amz-request-id",
		Destination: "http.x-amz-request-id",
		Type:        "cache",
		Action:      "delete",
		Priority:    uint(100),
	}

	log2 := gofastly.Header{
		Version:     1,
		Name:        "remove s3 server",
		Destination: "http.Server",
		Type:        "cache",
		Action:      "delete",
		IgnoreIfSet: true,
		Priority:    uint(100),
	}

	log3 := gofastly.Header{
		Version:     1,
		Name:        "DESTROY S3",
		Destination: "http.Server",
		Type:        "cache",
		Action:      "delete",
		Priority:    uint(100),
	}

	log4 := gofastly.Header{
		Version:           1,
		Name:              "Add server name",
		Destination:       "http.server-name",
		Type:              "request",
		Action:            "set",
		Source:            "server.identity",
		Priority:          uint(100),
		RequestCondition:  "test_req_condition",
		CacheCondition:    "test_cache_condition",
		ResponseCondition: "test_res_condition",
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckServiceV1Destroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccServiceV1HeadersConfig(name, domainName1),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckServiceV1Exists("fastly_service_v1.foo", &service),
					testAccCheckFastlyServiceV1HeaderAttributes(&service, []*gofastly.Header{&log1, &log2}),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "name", name),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "header.#", "2"),
				),
			},

			resource.TestStep{
				Config: testAccServiceV1HeadersConfig_update(name, domainName1),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckServiceV1Exists("fastly_service_v1.foo", &service),
					testAccCheckFastlyServiceV1HeaderAttributes(&service, []*gofastly.Header{&log1, &log3, &log4}),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "name", name),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "header.#", "3"),
				),
			},
		},
	})
}

func testAccCheckFastlyServiceV1HeaderAttributes(service *gofastly.ServiceDetail, headers []*gofastly.Header) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		conn := testAccProvider.Meta().(*FastlyClient).conn
		headersList, err := conn.ListHeaders(&gofastly.ListHeadersInput{
			Service: service.ID,
			Version: service.ActiveVersion.Number,
		})

		if err != nil {
			return fmt.Errorf("[ERR] Error looking up Headers for (%s), version (%v): %s", service.Name, service.ActiveVersion.Number, err)
		}

		if len(headersList) != len(headers) {
			return fmt.Errorf("Healthcheck List count mismatch, expected (%d), got (%d)", len(headers), len(headersList))
		}

		var found int
		for _, h := range headers {
			for _, lh := range headersList {
				if h.Name == lh.Name {
					// we don't know these things ahead of time, so populate them now
					h.ServiceID = service.ID
					h.Version = service.ActiveVersion.Number
					if !reflect.DeepEqual(h, lh) {
						return fmt.Errorf("Bad match Header match, expected (%#v), got (%#v)", h, lh)
					}
					found++
				}
			}
		}

		if found != len(headers) {
			return fmt.Errorf("Error matching Header rules")
		}

		return nil
	}
}

func testAccServiceV1HeadersConfig(name, domain string) string {
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

  header {
    destination = "http.x-amz-request-id"
    type        = "cache"
    action      = "delete"
    name        = "remove x-amz-request-id"
  }

  header {
    destination   = "http.Server"
    type          = "cache"
    action        = "delete"
    name          = "remove s3 server"
    ignore_if_set = "true"
  }

  force_destroy = true
}`, name, domain)
}

func testAccServiceV1HeadersConfig_update(name, domain string) string {
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

  header {
    destination = "http.x-amz-request-id"
    type        = "cache"
    action      = "delete"
    name        = "remove x-amz-request-id"
  }

  header {
    destination = "http.Server"
    type        = "cache"
    action      = "delete"
    name        = "DESTROY S3"
  }

	condition {
    name      = "test_req_condition"
    type      = "REQUEST"
    priority  = 5
    statement = "req.url ~ \"^/foo/bar$\""
  }

	condition {
    name      = "test_cache_condition"
    type      = "CACHE"
    priority  = 9
    statement = "req.url ~ \"^/articles/\""
  }

	condition {
    name      = "test_res_condition"
    type      = "RESPONSE"
    priority  = 10
    statement = "resp.status == 404"
  }

  header {
    destination 			 = "http.server-name"
    type        			 = "request"
    action      			 = "set"
    source      			 = "server.identity"
    name        			 = "Add server name"
		request_condition  = "test_req_condition"
		cache_condition    = "test_cache_condition"
		response_condition = "test_res_condition"
  }

  force_destroy = true
}`, name, domain)
}
