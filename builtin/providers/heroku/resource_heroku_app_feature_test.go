package heroku

import (
	"context"
	"fmt"
	"testing"

	heroku "github.com/cyberdelia/heroku-go/v3"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccHerokuAppFeature(t *testing.T) {
	var feature heroku.AppFeature
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuFeatureDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuFeature_basic(appName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuFeatureExists("heroku_app_feature.runtime_metrics", &feature),
					testAccCheckHerokuFeatureEnabled(&feature, true),
					resource.TestCheckResourceAttr(
						"heroku_app_feature.runtime_metrics", "enabled", "true",
					),
				),
			},
			{
				Config: testAccCheckHerokuFeature_disabled(appName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuFeatureExists("heroku_app_feature.runtime_metrics", &feature),
					testAccCheckHerokuFeatureEnabled(&feature, false),
					resource.TestCheckResourceAttr(
						"heroku_app_feature.runtime_metrics", "enabled", "false",
					),
				),
			},
		},
	})
}

func testAccCheckHerokuFeatureDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*heroku.Service)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "heroku_app_feature" {
			continue
		}

		_, err := client.AppFeatureInfo(context.TODO(), rs.Primary.Attributes["app"], rs.Primary.ID)

		if err == nil {
			return fmt.Errorf("Feature still exists")
		}
	}

	return nil
}

func testAccCheckHerokuFeatureExists(n string, feature *heroku.AppFeature) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No feature ID is set")
		}

		app, id := parseCompositeID(rs.Primary.ID)
		if app != rs.Primary.Attributes["app"] {
			return fmt.Errorf("Bad app: %s", app)
		}

		client := testAccProvider.Meta().(*heroku.Service)

		foundFeature, err := client.AppFeatureInfo(context.TODO(), app, id)
		if err != nil {
			return err
		}

		if foundFeature.ID != id {
			return fmt.Errorf("Feature not found")
		}

		*feature = *foundFeature
		return nil
	}
}

func testAccCheckHerokuFeatureEnabled(feature *heroku.AppFeature, enabled bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if feature.Enabled != enabled {
			return fmt.Errorf("Bad enabled: %v", feature.Enabled)
		}

		return nil
	}
}

func testAccCheckHerokuFeature_basic(appName string) string {
	return fmt.Sprintf(`
resource "heroku_app" "example" {
	name = "%s"
	region = "us"
}

resource "heroku_app_feature" "runtime_metrics" {
	app = "${heroku_app.example.name}"
	name = "log-runtime-metrics"
}
`, appName)
}

func testAccCheckHerokuFeature_disabled(appName string) string {
	return fmt.Sprintf(`
resource "heroku_app" "example" {
	name = "%s"
	region = "us"
}

resource "heroku_app_feature" "runtime_metrics" {
	app = "${heroku_app.example.name}"
	name = "log-runtime-metrics"
	enabled = false
}
`, appName)
}
