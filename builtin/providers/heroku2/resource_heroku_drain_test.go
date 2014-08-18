package heroku

import (
	"fmt"
	"testing"

	"github.com/bgentry/heroku-go"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccHerokuDrain_Basic(t *testing.T) {
	var drain heroku.LogDrain

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuDrainDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckHerokuDrainConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuDrainExists("heroku_drain.foobar", &drain),
					testAccCheckHerokuDrainAttributes(&drain),
					resource.TestCheckResourceAttr(
						"heroku_drain.foobar", "url", "syslog://terraform.example.com:1234"),
					resource.TestCheckResourceAttr(
						"heroku_drain.foobar", "app", "terraform-test-app"),
				),
			},
		},
	})
}

func testAccCheckHerokuDrainDestroy(s *terraform.State) error {
	client := testAccProvider.client

	for _, rs := range s.Resources {
		if rs.Type != "heroku_drain" {
			continue
		}

		_, err := client.LogDrainInfo(rs.Attributes["app"], rs.ID)

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
		rs, ok := s.Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.ID == "" {
			return fmt.Errorf("No Drain ID is set")
		}

		client := testAccProvider.client

		foundDrain, err := client.LogDrainInfo(rs.Attributes["app"], rs.ID)

		if err != nil {
			return err
		}

		if foundDrain.Id != rs.ID {
			return fmt.Errorf("Drain not found")
		}

		*Drain = *foundDrain

		return nil
	}
}

const testAccCheckHerokuDrainConfig_basic = `
resource "heroku_app" "foobar" {
    name = "terraform-test-app"
}

resource "heroku_drain" "foobar" {
    app = "${heroku_app.foobar.name}"
    url = "syslog://terraform.example.com:1234"
}`
