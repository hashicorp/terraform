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

func TestAccFastlyServiceV1_papertrail_basic(t *testing.T) {
	var service gofastly.ServiceDetail
	name := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	domainName1 := fmt.Sprintf("%s.notadomain.com", acctest.RandString(10))

	log1 := gofastly.Papertrail{
		Version:           1,
		Name:              "papertrailtesting",
		Address:           "test1.papertrailapp.com",
		Port:              uint(3600),
		Format:            "%h %l %u %t %r %>s",
		ResponseCondition: "test_response_condition",
	}

	log2 := gofastly.Papertrail{
		Version: 1,
		Name:    "papertrailtesting2",
		Address: "test2.papertrailapp.com",
		Port:    uint(8080),
		Format:  "%h %l %u %t %r %>s",
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckServiceV1Destroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccServiceV1PapertrailConfig(name, domainName1),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckServiceV1Exists("fastly_service_v1.foo", &service),
					testAccCheckFastlyServiceV1PapertrailAttributes(&service, []*gofastly.Papertrail{&log1}),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "name", name),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "papertrail.#", "1"),
				),
			},

			resource.TestStep{
				Config: testAccServiceV1PapertrailConfig_update(name, domainName1),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckServiceV1Exists("fastly_service_v1.foo", &service),
					testAccCheckFastlyServiceV1PapertrailAttributes(&service, []*gofastly.Papertrail{&log1, &log2}),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "name", name),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "papertrail.#", "2"),
				),
			},
		},
	})
}

func testAccCheckFastlyServiceV1PapertrailAttributes(service *gofastly.ServiceDetail, papertrails []*gofastly.Papertrail) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		conn := testAccProvider.Meta().(*FastlyClient).conn
		papertrailList, err := conn.ListPapertrails(&gofastly.ListPapertrailsInput{
			Service: service.ID,
			Version: service.ActiveVersion.Number,
		})

		if err != nil {
			return fmt.Errorf("[ERR] Error looking up Papertrail for (%s), version (%v): %s", service.Name, service.ActiveVersion.Number, err)
		}

		if len(papertrailList) != len(papertrails) {
			return fmt.Errorf("Papertrail List count mismatch, expected (%d), got (%d)", len(papertrails), len(papertrailList))
		}

		var found int
		for _, p := range papertrails {
			for _, lp := range papertrailList {
				if p.Name == lp.Name {
					// we don't know these things ahead of time, so populate them now
					p.ServiceID = service.ID
					p.Version = service.ActiveVersion.Number
					// We don't track these, so clear them out because we also wont know
					// these ahead of time
					lp.CreatedAt = nil
					lp.UpdatedAt = nil
					if !reflect.DeepEqual(p, lp) {
						return fmt.Errorf("Bad match Papertrail match, expected (%#v), got (%#v)", p, lp)
					}
					found++
				}
			}
		}

		if found != len(papertrails) {
			return fmt.Errorf("Error matching Papertrail rules")
		}

		return nil
	}
}

func testAccServiceV1PapertrailConfig(name, domain string) string {
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
    name      = "test_response_condition"
    type      = "RESPONSE"
    priority  = 5
    statement = "resp.status >= 400 && resp.status < 600"
  }

  papertrail {
    name               = "papertrailtesting"
    address            = "test1.papertrailapp.com"
    port               = 3600
		response_condition = "test_response_condition"
  }

  force_destroy = true
}`, name, domain)
}

func testAccServiceV1PapertrailConfig_update(name, domain string) string {
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
    name      = "test_response_condition"
    type      = "RESPONSE"
    priority  = 5
    statement = "resp.status >= 400 && resp.status < 600"
  }

	papertrail {
    name               = "papertrailtesting"
    address            = "test1.papertrailapp.com"
    port               = 3600
		response_condition = "test_response_condition"
  }

	papertrail {
    name               = "papertrailtesting2"
    address            = "test2.papertrailapp.com"
    port               = 8080
  }

  force_destroy = true
}`, name, domain)
}
