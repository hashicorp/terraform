package fastly

import (
	"fmt"
	"reflect"
	"sort"
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
				IgnoreIfSet: true,
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

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckServiceV1Destroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccServiceV1HeadersConfig(name, domainName1),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckServiceV1Exists("fastly_service_v1.foo", &service),
					testAccCheckFastlyServiceV1HeaderAttributes(&service, name, []string{"http.x-amz-request-id", "http.Server"}, nil),
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
					testAccCheckFastlyServiceV1HeaderAttributes(&service, name, []string{"http.x-amz-request-id", "http.Server"}, []string{"http.server-name"}),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "name", name),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "header.#", "3"),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "header.1147514417.source", "server.identity"),
				),
			},
		},
	})
}

func testAccCheckFastlyServiceV1HeaderAttributes(service *gofastly.ServiceDetail, name string, headersDeleted, headersAdded []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if service.Name != name {
			return fmt.Errorf("Bad name, expected (%s), got (%s)", name, service.Name)
		}

		conn := testAccProvider.Meta().(*FastlyClient).conn
		headersList, err := conn.ListHeaders(&gofastly.ListHeadersInput{
			Service: service.ID,
			Version: service.ActiveVersion.Number,
		})

		if err != nil {
			return fmt.Errorf("[ERR] Error looking up Headers for (%s), version (%s): %s", service.Name, service.ActiveVersion.Number, err)
		}

		var deleted []string
		var added []string
		for _, h := range headersList {
			if h.Action == gofastly.HeaderActionDelete {
				deleted = append(deleted, h.Destination)
			}
			if h.Action == gofastly.HeaderActionSet {
				added = append(added, h.Destination)
			}
		}

		sort.Strings(headersAdded)
		sort.Strings(headersDeleted)
		sort.Strings(deleted)
		sort.Strings(added)

		if !reflect.DeepEqual(headersDeleted, deleted) {
			return fmt.Errorf("Deleted Headers did not match.\n\tExpected: (%#v)\n\tGot: (%#v)", headersDeleted, deleted)
		}
		if !reflect.DeepEqual(headersAdded, added) {
			return fmt.Errorf("Added Headers did not match.\n\tExpected: (%#v)\n\tGot: (%#v)", headersAdded, added)
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
    destination = "http.Server"
    type        = "cache"
    action      = "delete"
    name        = "remove s3 server"
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

  header {
    destination = "http.server-name"
    type        = "request"
    action      = "set"
    source      = "server.identity"
    name        = "Add server name"
  }

  force_destroy = true
}`, name, domain)
}
