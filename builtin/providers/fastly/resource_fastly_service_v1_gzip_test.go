package fastly

import (
	"fmt"
	"reflect"
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

	log1 := gofastly.Gzip{
		Version:        1,
		Name:           "gzip file types",
		Extensions:     "js css",
		CacheCondition: "testing_condition",
	}

	log2 := gofastly.Gzip{
		Version:      1,
		Name:         "gzip extensions",
		ContentTypes: "text/css text/html",
	}

	log3 := gofastly.Gzip{
		Version:      1,
		Name:         "all",
		Extensions:   "js html css",
		ContentTypes: "text/javascript application/x-javascript application/javascript text/css text/html",
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckServiceV1Destroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccServiceV1GzipsConfig(name, domainName1),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckServiceV1Exists("fastly_service_v1.foo", &service),
					testAccCheckFastlyServiceV1GzipsAttributes(&service, []*gofastly.Gzip{&log1, &log2}),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "name", name),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "gzip.#", "2"),
				),
			},

			resource.TestStep{
				Config: testAccServiceV1GzipsConfig_update(name, domainName1),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckServiceV1Exists("fastly_service_v1.foo", &service),
					testAccCheckFastlyServiceV1GzipsAttributes(&service, []*gofastly.Gzip{&log3}),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "name", name),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "gzip.#", "1"),
				),
			},
		},
	})
}

func testAccCheckFastlyServiceV1GzipsAttributes(service *gofastly.ServiceDetail, gzips []*gofastly.Gzip) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		conn := testAccProvider.Meta().(*FastlyClient).conn
		gzipsList, err := conn.ListGzips(&gofastly.ListGzipsInput{
			Service: service.ID,
			Version: service.ActiveVersion.Number,
		})

		if err != nil {
			return fmt.Errorf("[ERR] Error looking up Gzips for (%s), version (%v): %s", service.Name, service.ActiveVersion.Number, err)
		}

		if len(gzipsList) != len(gzips) {
			return fmt.Errorf("Gzip count mismatch, expected (%d), got (%d)", len(gzips), len(gzipsList))
		}

		var found int
		for _, g := range gzips {
			for _, lg := range gzipsList {
				if g.Name == lg.Name {
					// we don't know these things ahead of time, so populate them now
					g.ServiceID = service.ID
					g.Version = service.ActiveVersion.Number
					if !reflect.DeepEqual(g, lg) {
						return fmt.Errorf("Bad match Gzip match, expected (%#v), got (%#v)", g, lg)
					}
					found++
				}
			}
		}

		if found != len(gzips) {
			return fmt.Errorf("Error matching Gzip rules")
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

	condition {
    name      = "testing_condition"
    type      = "CACHE"
    priority  = 10
    statement = "req.url ~ \"^/articles/\""
  }

  gzip {
    name       			= "gzip file types"
    extensions 			= ["css", "js"]
		cache_condition = "testing_condition"
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
