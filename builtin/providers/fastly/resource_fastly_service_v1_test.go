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

func TestResourceFastlyFlattenDomains(t *testing.T) {
	cases := []struct {
		remote []*gofastly.Domain
		local  []map[string]interface{}
	}{
		{
			remote: []*gofastly.Domain{
				&gofastly.Domain{
					Name:    "test.notexample.com",
					Comment: "not comment",
				},
			},
			local: []map[string]interface{}{
				map[string]interface{}{
					"name":    "test.notexample.com",
					"comment": "not comment",
				},
			},
		},
		{
			remote: []*gofastly.Domain{
				&gofastly.Domain{
					Name: "test.notexample.com",
				},
			},
			local: []map[string]interface{}{
				map[string]interface{}{
					"name":    "test.notexample.com",
					"comment": "",
				},
			},
		},
	}

	for _, c := range cases {
		out := flattenDomains(c.remote)
		if !reflect.DeepEqual(out, c.local) {
			t.Fatalf("Error matching:\nexpected: %#v\ngot: %#v", c.local, out)
		}
	}
}

func TestResourceFastlyFlattenBackend(t *testing.T) {
	cases := []struct {
		remote []*gofastly.Backend
		local  []map[string]interface{}
	}{
		{
			remote: []*gofastly.Backend{
				&gofastly.Backend{
					Name:                "test.notexample.com",
					Address:             "www.notexample.com",
					Port:                uint(80),
					AutoLoadbalance:     true,
					BetweenBytesTimeout: uint(10000),
					ConnectTimeout:      uint(1000),
					ErrorThreshold:      uint(0),
					FirstByteTimeout:    uint(15000),
					MaxConn:             uint(200),
					SSLCheckCert:        true,
					Weight:              uint(100),
				},
			},
			local: []map[string]interface{}{
				map[string]interface{}{
					"name":                  "test.notexample.com",
					"address":               "www.notexample.com",
					"port":                  80,
					"auto_loadbalance":      true,
					"between_bytes_timeout": 10000,
					"connect_timeout":       1000,
					"error_threshold":       0,
					"first_byte_timeout":    15000,
					"max_conn":              200,
					"ssl_check_cert":        true,
					"weight":                100,
				},
			},
		},
	}

	for _, c := range cases {
		out := flattenBackends(c.remote)
		if !reflect.DeepEqual(out, c.local) {
			t.Fatalf("Error matching:\nexpected: %#v\ngot: %#v", c.local, out)
		}
	}
}

func TestAccFastlyServiceV1_updateDomain(t *testing.T) {
	var service gofastly.ServiceDetail
	name := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	nameUpdate := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	domainName1 := fmt.Sprintf("%s.notadomain.com", acctest.RandString(10))
	domainName2 := fmt.Sprintf("%s.notadomain.com", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckServiceV1Destroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccServiceV1Config(name, domainName1),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckServiceV1Exists("fastly_service_v1.foo", &service),
					testAccCheckFastlyServiceV1Attributes(&service, name, []string{domainName1}),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "name", name),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "active_version", "1"),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "domain.#", "1"),
				),
			},

			resource.TestStep{
				Config: testAccServiceV1Config_domainUpdate(nameUpdate, domainName1, domainName2),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckServiceV1Exists("fastly_service_v1.foo", &service),
					testAccCheckFastlyServiceV1Attributes(&service, nameUpdate, []string{domainName1, domainName2}),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "name", nameUpdate),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "active_version", "2"),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "domain.#", "2"),
				),
			},
		},
	})
}

func TestAccFastlyServiceV1_updateBackend(t *testing.T) {
	var service gofastly.ServiceDetail
	name := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	backendName := fmt.Sprintf("%s.aws.amazon.com", acctest.RandString(3))
	backendName2 := fmt.Sprintf("%s.aws.amazon.com", acctest.RandString(3))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckServiceV1Destroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccServiceV1Config_backend(name, backendName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckServiceV1Exists("fastly_service_v1.foo", &service),
					testAccCheckFastlyServiceV1Attributes_backends(&service, name, []string{backendName}),
				),
			},

			resource.TestStep{
				Config: testAccServiceV1Config_backend_update(name, backendName, backendName2),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckServiceV1Exists("fastly_service_v1.foo", &service),
					testAccCheckFastlyServiceV1Attributes_backends(&service, name, []string{backendName, backendName2}),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "active_version", "2"),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "backend.#", "2"),
				),
			},
		},
	})
}

func TestAccFastlyServiceV1_basic(t *testing.T) {
	var service gofastly.ServiceDetail
	name := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	domainName := fmt.Sprintf("%s.notadomain.com", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckServiceV1Destroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccServiceV1Config(name, domainName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckServiceV1Exists("fastly_service_v1.foo", &service),
					testAccCheckFastlyServiceV1Attributes(&service, name, []string{domainName}),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "name", name),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "active_version", "1"),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "domain.#", "1"),
				),
			},
		},
	})
}

// ServiceV1_disappears – test that a non-empty plan is returned when a Fastly
// Service is destroyed outside of Terraform, and can no longer be found,
// correctly clearing the ID field and generating a new plan
func TestAccFastlyServiceV1_disappears(t *testing.T) {
	var service gofastly.ServiceDetail
	name := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	domainName := fmt.Sprintf("%s.notadomain.com", acctest.RandString(10))

	testDestroy := func(*terraform.State) error {
		// reach out and DELETE the service
		conn := testAccProvider.Meta().(*FastlyClient).conn
		// deactivate active version to destoy
		_, err := conn.DeactivateVersion(&gofastly.DeactivateVersionInput{
			Service: service.ID,
			Version: service.ActiveVersion.Number,
		})
		if err != nil {
			return err
		}

		// delete service
		err = conn.DeleteService(&gofastly.DeleteServiceInput{
			ID: service.ID,
		})

		if err != nil {
			return err
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckServiceV1Destroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccServiceV1Config(name, domainName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckServiceV1Exists("fastly_service_v1.foo", &service),
					testDestroy,
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccCheckServiceV1Exists(n string, service *gofastly.ServiceDetail) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Service ID is set")
		}

		conn := testAccProvider.Meta().(*FastlyClient).conn
		latest, err := conn.GetServiceDetails(&gofastly.GetServiceInput{
			ID: rs.Primary.ID,
		})

		if err != nil {
			return err
		}

		*service = *latest

		return nil
	}
}

func testAccCheckFastlyServiceV1Attributes(service *gofastly.ServiceDetail, name string, domains []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if service.Name != name {
			return fmt.Errorf("Bad name, expected (%s), got (%s)", name, service.Name)
		}

		conn := testAccProvider.Meta().(*FastlyClient).conn
		domainList, err := conn.ListDomains(&gofastly.ListDomainsInput{
			Service: service.ID,
			Version: service.ActiveVersion.Number,
		})

		if err != nil {
			return fmt.Errorf("[ERR] Error looking up Domains for (%s), version (%s): %s", service.Name, service.ActiveVersion.Number, err)
		}

		expected := len(domains)
		for _, d := range domainList {
			for _, e := range domains {
				if d.Name == e {
					expected--
				}
			}
		}

		if expected > 0 {
			return fmt.Errorf("Domain count mismatch, expected: %#v, got: %#v", domains, domainList)
		}

		return nil
	}
}

func testAccCheckFastlyServiceV1Attributes_backends(service *gofastly.ServiceDetail, name string, backends []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if service.Name != name {
			return fmt.Errorf("Bad name, expected (%s), got (%s)", name, service.Name)
		}

		conn := testAccProvider.Meta().(*FastlyClient).conn
		backendList, err := conn.ListBackends(&gofastly.ListBackendsInput{
			Service: service.ID,
			Version: service.ActiveVersion.Number,
		})

		if err != nil {
			return fmt.Errorf("[ERR] Error looking up Backends for (%s), version (%s): %s", service.Name, service.ActiveVersion.Number, err)
		}

		expected := len(backendList)
		for _, b := range backendList {
			for _, e := range backends {
				if b.Address == e {
					expected--
				}
			}
		}

		if expected > 0 {
			return fmt.Errorf("Backend count mismatch, expected: %#v, got: %#v", backends, backendList)
		}

		return nil
	}
}

func testAccCheckServiceV1Destroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "fastly_service_v1" {
			continue
		}

		conn := testAccProvider.Meta().(*FastlyClient).conn
		l, err := conn.ListServices(&gofastly.ListServicesInput{})
		if err != nil {
			return fmt.Errorf("[WARN] Error listing servcies when deleting Fastly Service (%s): %s", rs.Primary.ID, err)
		}

		for _, s := range l {
			if s.ID == rs.Primary.ID {
				// service still found
				return fmt.Errorf("[WARN] Tried deleting Service (%s), but was still found", rs.Primary.ID)
			}
		}
	}
	return nil
}

func testAccServiceV1Config(name, domain string) string {
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

  force_destroy = true
}`, name, domain)
}

func testAccServiceV1Config_domainUpdate(name, domain1, domain2 string) string {
	return fmt.Sprintf(`
resource "fastly_service_v1" "foo" {
  name = "%s"

  domain {
    name    = "%s"
    comment = "tf-testing-domain"
  }

  domain {
    name    = "%s"
    comment = "tf-testing-other-domain"
  }

  backend {
    address = "aws.amazon.com"
    name    = "amazon docs"
  }

  force_destroy = true
}`, name, domain1, domain2)
}

func testAccServiceV1Config_backend(name, backend string) string {
	return fmt.Sprintf(`
resource "fastly_service_v1" "foo" {
  name = "%s"

  domain {
    name    = "test.notadomain.com"
    comment = "tf-testing-domain"
  }

  backend {
    address = "%s"
    name    = "tf -test backend"
  }

  force_destroy = true
}`, name, backend)
}

func testAccServiceV1Config_backend_update(name, backend, backend2 string) string {
	return fmt.Sprintf(`
resource "fastly_service_v1" "foo" {
  name = "%s"

	default_ttl = 3400

  domain {
    name    = "test.notadomain.com"
    comment = "tf-testing-domain"
  }

  backend {
    address = "%s"
    name    = "tf-test-backend"
  }

  backend {
    address = "%s"
    name    = "tf-test-backend-other"
  }

  force_destroy = true
}`, name, backend, backend2)
}
