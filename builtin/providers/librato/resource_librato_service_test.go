package librato

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/henrikhodne/go-librato/librato"
)

func TestAccLibratoService_Basic(t *testing.T) {
	var service librato.Service

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLibratoServiceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckLibratoServiceConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckLibratoServiceExists("librato_service.foobar", &service),
					testAccCheckLibratoServiceTitle(&service, "Foo Bar"),
					resource.TestCheckResourceAttr(
						"librato_service.foobar", "title", "Foo Bar"),
				),
			},
		},
	})
}

func TestAccLibratoService_Updated(t *testing.T) {
	var service librato.Service

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLibratoServiceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckLibratoServiceConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckLibratoServiceExists("librato_service.foobar", &service),
					testAccCheckLibratoServiceTitle(&service, "Foo Bar"),
					resource.TestCheckResourceAttr(
						"librato_service.foobar", "title", "Foo Bar"),
				),
			},
			resource.TestStep{
				Config: testAccCheckLibratoServiceConfig_new_value,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckLibratoServiceExists("librato_service.foobar", &service),
					testAccCheckLibratoServiceTitle(&service, "Bar Baz"),
					resource.TestCheckResourceAttr(
						"librato_service.foobar", "title", "Bar Baz"),
				),
			},
		},
	})
}

func testAccCheckLibratoServiceDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*librato.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "librato_service" {
			continue
		}

		id, err := strconv.ParseUint(rs.Primary.ID, 10, 0)
		if err != nil {
			return fmt.Errorf("ID not a number")
		}

		_, _, err = client.Services.Get(uint(id))

		if err == nil {
			return fmt.Errorf("Service still exists")
		}
	}

	return nil
}

func testAccCheckLibratoServiceTitle(service *librato.Service, title string) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if service.Title == nil || *service.Title != title {
			return fmt.Errorf("Bad title: %s", *service.Title)
		}

		return nil
	}
}

func testAccCheckLibratoServiceExists(n string, service *librato.Service) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Service ID is set")
		}

		client := testAccProvider.Meta().(*librato.Client)

		id, err := strconv.ParseUint(rs.Primary.ID, 10, 0)
		if err != nil {
			return fmt.Errorf("ID not a number")
		}

		foundService, _, err := client.Services.Get(uint(id))

		if err != nil {
			return err
		}

		if foundService.ID == nil || *foundService.ID != uint(id) {
			return fmt.Errorf("Service not found")
		}

		*service = *foundService

		return nil
	}
}

const testAccCheckLibratoServiceConfig_basic = `
resource "librato_service" "foobar" {
    title = "Foo Bar"
    type = "mail"
    settings = <<EOF
{
  "addresses": "admin@example.com"
}
EOF
}`

const testAccCheckLibratoServiceConfig_new_value = `
resource "librato_service" "foobar" {
    title = "Bar Baz"
    type = "mail"
    settings = <<EOF
{
  "addresses": "admin@example.com"
}
EOF
}`
