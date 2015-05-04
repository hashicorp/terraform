package heroku

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/cyberdelia/heroku-go/v3"
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
					testAccCheckHerokuAddonAttributes(&addon, "deployhooks:http"),
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

// GH-198
func TestAccHerokuAddon_noPlan(t *testing.T) {
	var addon heroku.Addon

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuAddonDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckHerokuAddonConfig_no_plan,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuAddonExists("heroku_addon.foobar", &addon),
					testAccCheckHerokuAddonAttributes(&addon, "memcachier:dev"),
					resource.TestCheckResourceAttr(
						"heroku_addon.foobar", "app", "terraform-test-app"),
					resource.TestCheckResourceAttr(
						"heroku_addon.foobar", "plan", "memcachier"),
					testAccCheckHerokuAddonConfigVars("heroku_addon.foobar", []string{"MEMCACHIER_SERVERS", "MEMCACHIER_USERNAME", "MEMCACHIER_PASSWORD"}),
				),
			},
			resource.TestStep{
				Config: testAccCheckHerokuAddonConfig_no_plan,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuAddonExists("heroku_addon.foobar", &addon),
					testAccCheckHerokuAddonAttributes(&addon, "memcachier:dev"),
					resource.TestCheckResourceAttr(
						"heroku_addon.foobar", "app", "terraform-test-app"),
					resource.TestCheckResourceAttr(
						"heroku_addon.foobar", "plan", "memcachier"),
					testAccCheckHerokuAddonConfigVars("heroku_addon.foobar", []string{"MEMCACHIER_SERVERS", "MEMCACHIER_USERNAME", "MEMCACHIER_PASSWORD"}),
				),
			},
		},
	})
}

func testAccCheckHerokuAddonConfigVars(addonName string, vars []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		instanceState := s.RootModule().Resources[addonName].Primary
		definedVarsMap := make(map[string]bool)
		for i := range vars {
			definedVarsMap[vars[i]] = false
		}

		size, _ := strconv.Atoi(instanceState.Attributes["config_vars.#"])
		for i := 0; i < size; i++ {
			configVar := instanceState.Attributes["config_vars."+fmt.Sprintf("%d", i)]
			definedVarsMap[configVar] = true
		}

		for k := range definedVarsMap {
			if !definedVarsMap[k] {
				return fmt.Errorf("Desired config variable %s not set in addon state", k)
			}
		}

		return nil
	}
}

func testAccCheckHerokuAddonDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*heroku.Service)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "heroku_addon" {
			continue
		}

		_, err := client.AddonInfo(rs.Primary.Attributes["app"], rs.Primary.ID)

		if err == nil {
			return fmt.Errorf("Addon still exists")
		}
	}

	return nil
}

func testAccCheckHerokuAddonAttributes(addon *heroku.Addon, plan string) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if addon.Plan.Name != plan {
			return fmt.Errorf("Bad plan: %s", addon.Plan.Name)
		}

		return nil
	}
}

func testAccCheckHerokuAddonExists(n string, addon *heroku.Addon) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Addon ID is set")
		}

		client := testAccProvider.Meta().(*heroku.Service)

		foundAddon, err := client.AddonInfo(rs.Primary.Attributes["app"], rs.Primary.ID)

		if err != nil {
			return err
		}

		if foundAddon.ID != rs.Primary.ID {
			return fmt.Errorf("Addon not found")
		}

		*addon = *foundAddon

		return nil
	}
}

const testAccCheckHerokuAddonConfig_basic = `
resource "heroku_app" "foobar" {
    name = "terraform-test-app"
    region = "us"
}

resource "heroku_addon" "foobar" {
    app = "${heroku_app.foobar.name}"
    plan = "deployhooks:http"
    config {
        url = "http://google.com"
    }
}`

const testAccCheckHerokuAddonConfig_no_plan = `
resource "heroku_app" "foobar" {
    name = "terraform-test-app"
    region = "us"
}

resource "heroku_addon" "foobar" {
    app = "${heroku_app.foobar.name}"
    plan = "memcachier"
}`
