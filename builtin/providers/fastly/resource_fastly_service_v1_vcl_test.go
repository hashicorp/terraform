package fastly

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	gofastly "github.com/sethvargo/go-fastly"
)

func TestAccFastlyServiceV1_VCL_basic(t *testing.T) {
	var service gofastly.ServiceDetail
	name := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	domainName1 := fmt.Sprintf("%s.notadomain.com", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckServiceV1Destroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccServiceV1VCLConfig(name, domainName1),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckServiceV1Exists("fastly_service_v1.foo", &service),
					testAccCheckFastlyServiceV1VCLAttributes(&service, name, 1),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "name", name),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "vcl.#", "1"),
				),
			},

			resource.TestStep{
				Config: testAccServiceV1VCLConfig_update(name, domainName1),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckServiceV1Exists("fastly_service_v1.foo", &service),
					testAccCheckFastlyServiceV1VCLAttributes(&service, name, 2),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "name", name),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "vcl.#", "2"),
				),
			},
		},
	})
}

func testAccCheckFastlyServiceV1VCLAttributes(service *gofastly.ServiceDetail, name string, vclCount int) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if service.Name != name {
			return fmt.Errorf("Bad name, expected (%s), got (%s)", name, service.Name)
		}

		conn := testAccProvider.Meta().(*FastlyClient).conn
		vclList, err := conn.ListVCLs(&gofastly.ListVCLsInput{
			Service: service.ID,
			Version: service.ActiveVersion.Number,
		})

		if err != nil {
			return fmt.Errorf("[ERR] Error looking up VCL for (%s), version (%v): %s", service.Name, service.ActiveVersion.Number, err)
		}

		if len(vclList) != vclCount {
			return fmt.Errorf("VCL count mismatch, expected (%d), got (%d)", vclCount, len(vclList))
		}

		return nil
	}
}

func testAccServiceV1VCLConfig(name, domain string) string {
	return fmt.Sprintf(`
resource "fastly_service_v1" "foo" {
  name = "%s"

  domain {
    name    = "%s"
    comment = "tf-testing-domain"
  }

  vcl {
    name    = "my_custom_main_vcl"
    content = <<EOF
sub vcl_recv {
#FASTLY recv

    if (req.request != "HEAD" && req.request != "GET" && req.request != "FASTLYPURGE") {
      return(pass);
    }

    return(lookup);
}

backend amazondocs {
  .host = "127.0.0.1";
  .port = "80";
}
EOF
    main    = true
  }

  force_destroy = true
}`, name, domain)
}

func testAccServiceV1VCLConfig_update(name, domain string) string {
	return fmt.Sprintf(`
resource "fastly_service_v1" "foo" {
  name = "%s"

  domain {
    name    = "%s"
    comment = "tf-testing-domain"
  }

  vcl {
    name    = "my_custom_main_vcl"
    content = <<EOF
sub vcl_recv {
#FASTLY recv

    if (req.request != "HEAD" && req.request != "GET" && req.request != "FASTLYPURGE") {
      return(pass);
    }

    return(lookup);
}

backend amazondocs {
  .host = "127.0.0.1";
  .port = "80";
}
EOF
    main    = true
  }

        vcl {
                name = "other_vcl"
                content = <<EOF
sub vcl_error {
#FASTLY error
}

backend amazondocs {
  .host = "127.0.0.1";
  .port = "80";
}
EOF
        }

  force_destroy = true
}`, name, domain)
}
