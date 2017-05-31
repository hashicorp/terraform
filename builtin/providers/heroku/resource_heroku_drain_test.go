package heroku

import (
	"context"
	"fmt"
	"testing"

	"github.com/cyberdelia/heroku-go/v3"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccHerokuDrain_Basic(t *testing.T) {
	var drain heroku.LogDrain
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuDrainDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuDrainConfig_basic(appName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuDrainExists("heroku_drain.foobar", &drain),
					testAccCheckHerokuDrainAttributes(&drain),
					resource.TestCheckResourceAttr(
						"heroku_drain.foobar", "url", "syslog://terraform.example.com:1234"),
					resource.TestCheckResourceAttr(
						"heroku_drain.foobar", "app", appName),
				),
			},
		},
	})
}

func testAccCheckHerokuDrainDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*heroku.Service)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "heroku_drain" {
			continue
		}

		_, err := client.LogDrainInfo(context.TODO(), rs.Primary.Attributes["app"], rs.Primary.ID)

		if err == nil {
			return fmt.Errorf("Drain still exists")
		}
	}

	return nil
}

func testAccCheckHerokuDrainAttributes(Drain *heroku.LogDrain) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if Drain.URL != "syslog://terraform.example.com:1234" {
			return fmt.Errorf("Bad URL: %s", Drain.URL)
		}

		if Drain.Token == "" {
			return fmt.Errorf("No token: %#v", Drain)
		}

		return nil
	}
}

func testAccCheckHerokuDrainExists(n string, Drain *heroku.LogDrain) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Drain ID is set")
		}

		client := testAccProvider.Meta().(*heroku.Service)

		foundDrain, err := client.LogDrainInfo(context.TODO(), rs.Primary.Attributes["app"], rs.Primary.ID)

		if err != nil {
			return err
		}

		if foundDrain.ID != rs.Primary.ID {
			return fmt.Errorf("Drain not found")
		}

		*Drain = *foundDrain

		return nil
	}
}

func testAccCheckHerokuDrainConfig_basic(appName string) string {
	return fmt.Sprintf(`
resource "heroku_app" "foobar" {
    name = "%s"
    region = "us"
}

resource "heroku_drain" "foobar" {
    app = "${heroku_app.foobar.name}"
    url = "syslog://terraform.example.com:1234"
}`, appName)
}
