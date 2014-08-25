package google

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccComputeAddress_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		//CheckDestroy: testAccCheckHerokuAppDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeAddress_basic,
				/*
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuAppExists("heroku_app.foobar", &app),
					testAccCheckHerokuAppAttributes(&app),
					resource.TestCheckResourceAttr(
						"heroku_app.foobar", "name", "terraform-test-app"),
					resource.TestCheckResourceAttr(
						"heroku_app.foobar", "config_vars.0.FOO", "bar"),
				),
				*/
			},
		},
	})
}

/*
func testAccCheckHerokuAppDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*heroku.Client)

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
		client := testAccProvider.Meta().(*heroku.Client)

		if app.Region.Name != "us" {
			return fmt.Errorf("Bad region: %s", app.Region.Name)
		}

		if app.Stack.Name != "cedar" {
			return fmt.Errorf("Bad stack: %s", app.Stack.Name)
		}

		if app.Name != "terraform-test-app" {
			return fmt.Errorf("Bad name: %s", app.Name)
		}

		vars, err := client.ConfigVarInfo(app.Name)
		if err != nil {
			return err
		}

		if vars["FOO"] != "bar" {
			return fmt.Errorf("Bad config vars: %v", vars)
		}

		return nil
	}
}

func testAccCheckHerokuAppAttributesUpdated(app *heroku.App) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*heroku.Client)

		if app.Name != "terraform-test-renamed" {
			return fmt.Errorf("Bad name: %s", app.Name)
		}

		vars, err := client.ConfigVarInfo(app.Name)
		if err != nil {
			return err
		}

		// Make sure we kept the old one
		if vars["FOO"] != "bing" {
			return fmt.Errorf("Bad config vars: %v", vars)
		}

		if vars["BAZ"] != "bar" {
			return fmt.Errorf("Bad config vars: %v", vars)
		}

		return nil

	}
}

func testAccCheckHerokuAppAttributesNoVars(app *heroku.App) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*heroku.Client)

		if app.Name != "terraform-test-app" {
			return fmt.Errorf("Bad name: %s", app.Name)
		}

		vars, err := client.ConfigVarInfo(app.Name)
		if err != nil {
			return err
		}

		if len(vars) != 0 {
			return fmt.Errorf("vars exist: %v", vars)
		}

		return nil
	}
}

func testAccCheckHerokuAppExists(n string, app *heroku.App) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.ID == "" {
			return fmt.Errorf("No App Name is set")
		}

		client := testAccProvider.Meta().(*heroku.Client)

		foundApp, err := client.AppInfo(rs.ID)

		if err != nil {
			return err
		}

		if foundApp.Name != rs.ID {
			return fmt.Errorf("App not found")
		}

		*app = *foundApp

		return nil
	}
}
*/

const testAccComputeAddress_basic = `
resource "google_compute_address" "foobar" {
	name = "terraform-test"
}`
