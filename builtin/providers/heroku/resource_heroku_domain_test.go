package heroku

import (
	"fmt"
	"testing"

	"github.com/cyberdelia/heroku-go/v3"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccHerokuDomain_Basic(t *testing.T) {
	var domain heroku.Domain

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuDomainDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckHerokuDomainConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuDomainExists("heroku_domain.foobar", &domain),
					testAccCheckHerokuDomainAttributes(&domain),
					resource.TestCheckResourceAttr(
						"heroku_domain.foobar", "hostname", "terraform.example.com"),
					resource.TestCheckResourceAttr(
						"heroku_domain.foobar", "app", "terraform-test-app"),
					resource.TestCheckResourceAttr(
						"heroku_domain.foobar", "cname", "terraform-test-app.herokuapp.com"),
				),
			},
		},
	})
}

func testAccCheckHerokuDomainDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*heroku.Service)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "heroku_domain" {
			continue
		}

		_, err := client.DomainInfo(rs.Primary.Attributes["app"], rs.Primary.ID)

		if err == nil {
			return fmt.Errorf("Domain still exists")
		}
	}

	return nil
}

func testAccCheckHerokuDomainAttributes(Domain *heroku.Domain) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if Domain.Hostname != "terraform.example.com" {
			return fmt.Errorf("Bad hostname: %s", Domain.Hostname)
		}

		return nil
	}
}

func testAccCheckHerokuDomainExists(n string, Domain *heroku.Domain) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Domain ID is set")
		}

		client := testAccProvider.Meta().(*heroku.Service)

		foundDomain, err := client.DomainInfo(rs.Primary.Attributes["app"], rs.Primary.ID)

		if err != nil {
			return err
		}

		if foundDomain.ID != rs.Primary.ID {
			return fmt.Errorf("Domain not found")
		}

		*Domain = *foundDomain

		return nil
	}
}

const testAccCheckHerokuDomainConfig_basic = `
resource "heroku_app" "foobar" {
    name = "terraform-test-app"
    region = "us"
}

resource "heroku_domain" "foobar" {
    app = "${heroku_app.foobar.name}"
    hostname = "terraform.example.com"
}`
