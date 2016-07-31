package fastly

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	gofastly "github.com/sethvargo/go-fastly"
)

func TestFastlyServiceV1_FlattenGzips(t *testing.T) {
	cases := []struct {
		remote []*gofastly.Gzip
		local  []map[string]interface{}
	}{
		{
			remote: []*gofastly.Gzip{
				&gofastly.Gzip{
					Name:       "somegzip",
					Extensions: "css",
				},
			},
			local: []map[string]interface{}{
				map[string]interface{}{
					"name":       "somegzip",
					"extensions": schema.NewSet(schema.HashString, []interface{}{"css"}),
				},
			},
		},
		{
			remote: []*gofastly.Gzip{
				&gofastly.Gzip{
					Name:         "somegzip",
					Extensions:   "css json js",
					ContentTypes: "text/html",
				},
				&gofastly.Gzip{
					Name:         "someothergzip",
					Extensions:   "css js",
					ContentTypes: "text/html text/xml",
				},
			},
			local: []map[string]interface{}{
				map[string]interface{}{
					"name":          "somegzip",
					"extensions":    schema.NewSet(schema.HashString, []interface{}{"css", "json", "js"}),
					"content_types": schema.NewSet(schema.HashString, []interface{}{"text/html"}),
				},
				map[string]interface{}{
					"name":          "someothergzip",
					"extensions":    schema.NewSet(schema.HashString, []interface{}{"css", "js"}),
					"content_types": schema.NewSet(schema.HashString, []interface{}{"text/html", "text/xml"}),
				},
			},
		},
	}

	for _, c := range cases {
		out := flattenGzips(c.remote)
		// loop, because deepequal wont work with our sets
		expectedCount := len(c.local)
		var found int
		for _, o := range out {
			for _, l := range c.local {
				if o["name"].(string) == l["name"].(string) {
					found++
					if o["extensions"] == nil && l["extensions"] != nil {
						t.Fatalf("output extensions are nil, local are not")
					}

					if o["extensions"] != nil {
						oex := o["extensions"].(*schema.Set)
						lex := l["extensions"].(*schema.Set)
						if !oex.Equal(lex) {
							t.Fatalf("Extensions don't match, expected: %#v, got: %#v", lex, oex)
						}
					}

					if o["content_types"] == nil && l["content_types"] != nil {
						t.Fatalf("output content types are nil, local are not")
					}

					if o["content_types"] != nil {
						oct := o["content_types"].(*schema.Set)
						lct := l["content_types"].(*schema.Set)
						if !oct.Equal(lct) {
							t.Fatalf("ContentTypes don't match, expected: %#v, got: %#v", lct, oct)
						}
					}

				}
			}
		}

		if found != expectedCount {
			t.Fatalf("Found and expected mismatch: %d / %d", found, expectedCount)
		}
	}
}

func TestAccFastlyServiceV1_gzips_basic(t *testing.T) {
	var service gofastly.ServiceDetail
	name := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	domainName1 := fmt.Sprintf("%s.notadomain.com", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckServiceV1Destroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccServiceV1GzipsConfig(name, domainName1),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckServiceV1Exists("fastly_service_v1.foo", &service),
					testAccCheckFastlyServiceV1GzipsAttributes(&service, name, 2),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "name", name),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "gzip.#", "2"),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "gzip.3704620722.extensions.#", "2"),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "gzip.3704620722.content_types.#", "0"),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "gzip.3820313126.content_types.#", "2"),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "gzip.3820313126.extensions.#", "0"),
				),
			},

			resource.TestStep{
				Config: testAccServiceV1GzipsConfig_update(name, domainName1),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckServiceV1Exists("fastly_service_v1.foo", &service),
					testAccCheckFastlyServiceV1GzipsAttributes(&service, name, 1),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "name", name),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "gzip.#", "1"),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "gzip.3694165387.extensions.#", "3"),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "gzip.3694165387.content_types.#", "5"),
				),
			},
		},
	})
}

func testAccCheckFastlyServiceV1GzipsAttributes(service *gofastly.ServiceDetail, name string, gzipCount int) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if service.Name != name {
			return fmt.Errorf("Bad name, expected (%s), got (%s)", name, service.Name)
		}

		conn := testAccProvider.Meta().(*FastlyClient).conn
		gzipsList, err := conn.ListGzips(&gofastly.ListGzipsInput{
			Service: service.ID,
			Version: service.ActiveVersion.Number,
		})

		if err != nil {
			return fmt.Errorf("[ERR] Error looking up Gzips for (%s), version (%s): %s", service.Name, service.ActiveVersion.Number, err)
		}

		if len(gzipsList) != gzipCount {
			return fmt.Errorf("Gzip count mismatch, expected (%d), got (%d)", gzipCount, len(gzipsList))
		}

		return nil
	}
}

func testAccServiceV1GzipsConfig(name, domain string) string {
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

  gzip {
    name       = "gzip file types"
    extensions = ["css", "js"]
  }

  gzip {
    name          = "gzip extensions"
    content_types = ["text/html", "text/css"]
  }

  force_destroy = true
}`, name, domain)
}

func testAccServiceV1GzipsConfig_update(name, domain string) string {
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

  gzip {
    name       = "all"
    extensions = ["css", "js", "html"]

    content_types = [
      "text/html",
      "text/css",
      "application/x-javascript",
      "text/css",
      "application/javascript",
      "text/javascript",
    ]
  }

  force_destroy = true
}`, name, domain)
}
