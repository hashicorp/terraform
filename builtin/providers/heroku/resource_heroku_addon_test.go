package heroku

import (
	"fmt"
	"testing"

	"github.com/bgentry/heroku-go"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccHerokuAddon_Basic(t *testing.T) {
	var addon heroku.Addon

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuAddonDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckHerokuAddonConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuAddonExists("heroku_addon.foobar", &addon),
					testAccCheckHerokuAddonAttributes(&addon),
					resource.TestCheckResourceAttr(
						"heroku_addon.foobar", "config.0.url", "http://google.com"),
					resource.TestCheckResourceAttr(
						"heroku_addon.foobar", "app", "terraform-test-app"),
					resource.TestCheckResourceAttr(
						"heroku_addon.foobar", "plan", "deployhooks:http"),
				),
			},
		},
	})
}

func testAccCheckHerokuAddonDestroy(s *terraform.State) error {
	client := testAccProvider.client

	for _, rs := range s.Resources {
		if rs.Type != "heroku_addon" {
			continue
		}

		_, err := client.AddonInfo(rs.Attributes["app"], rs.ID)

		if err == nil {
			return fmt.Errorf("Addon still exists")
		}
	}

	return nil
}

func testAccCheckHerokuAddonAttributes(addon *heroku.Addon) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if addon.Plan.Name != "deployhooks:http" {
			return fmt.Errorf("Bad plan: %s", addon.Plan)
		}

		return nil
	}
}

func testAccCheckHerokuAddonExists(n string, addon *heroku.Addon) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.ID == "" {
			return fmt.Errorf("No Addon ID is set")
		}

		client := testAccProvider.client

		foundAddon, err := client.AddonInfo(rs.Attributes["app"], rs.ID)

		if err != nil {
			return err
		}

		if foundAddon.Id != rs.ID {
			return fmt.Errorf("Addon not found")
		}

		*addon = *foundAddon

		return nil
	}
}

const testAccCheckHerokuAddonConfig_basic = `
resource "heroku_app" "foobar" {
    name = "terraform-test-app"
}

resource "heroku_addon" "foobar" {
    app = "${heroku_app.foobar.name}"
    plan = "deployhooks:http"
    config {
        url = "http://google.com"
    }
}`
