package heroku

import (
	"fmt"
	"testing"

	"github.com/bgentry/heroku-go"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccHerokuApp_Basic(t *testing.T) {
	var app heroku.App

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuAppDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckHerokuAppConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuAppExists("heroku_app.foobar", &app),
					testAccCheckHerokuAppAttributes(&app),
					resource.TestCheckResourceAttr(
						"heroku_app.foobar", "name", "terraform-test-app"),
				),
			},
		},
	})
}

func testAccCheckHerokuAppDestroy(s *terraform.State) error {
	client := testAccProvider.client

	for _, rs := range s.Resources {
		if rs.Type != "heroku_app" {
			continue
		}

		_, err := client.AppInfo(rs.ID)

		if err == nil {
			return fmt.Errorf("App still exists")
		}
	}

	return nil
}

func testAccCheckHerokuAppAttributes(app *heroku.App) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		// check attrs

		return nil
	}
}

func testAccCheckHerokuAppExists(n string, app *heroku.App) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.Resources[n]
		fmt.Printf("resources %#v", s.Resources)
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.ID == "" {
			return fmt.Errorf("No App Name is set")
		}

		client := testAccProvider.client

		foundApp, err := client.AppInfo(rs.ID)

		if err != nil {
			return err
		}

		if foundApp.Name != rs.ID {
			return fmt.Errorf("App not found")
		}

		app = foundApp

		return nil
	}
}

const testAccCheckHerokuAppConfig_basic = `
resource "heroku_app" "foobar" {
    name = "terraform-test-app"
}`
