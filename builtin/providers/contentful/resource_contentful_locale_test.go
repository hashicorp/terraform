package contentful

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	contentful "github.com/tolgaakyuz/contentful-go"
)

func TestAccContentfulLocales_Basic(t *testing.T) {
	var locale contentful.Locale

	spaceName := fmt.Sprintf("space-name-%s", acctest.RandString(3))
	name := fmt.Sprintf("locale-name-%s", acctest.RandString(3))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccContentfulLocaleDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccContentfulLocaleConfig(spaceName, name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckContentfulLocaleExists("contentful_locale.mylocale", &locale),
					testAccCheckContentfulLocaleAttributes(&locale, map[string]interface{}{
						"name":          name,
						"code":          "de",
						"fallback_code": "en-US",
						"optional":      false,
						"cda":           false,
						"cma":           true,
					}),
				),
			},
			resource.TestStep{
				Config: testAccContentfulLocaleUpdateConfig(spaceName, name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckContentfulLocaleExists("contentful_locale.mylocale", &locale),
					testAccCheckContentfulLocaleAttributes(&locale, map[string]interface{}{
						"name":          fmt.Sprintf("%s-updated", name),
						"code":          "es",
						"fallback_code": "en-US",
						"optional":      true,
						"cda":           true,
						"cma":           false,
					}),
				),
			},
		},
	})
}

func testAccCheckContentfulLocaleExists(n string, locale *contentful.Locale) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not Found: %s", n)
		}

		spaceID := rs.Primary.Attributes["space_id"]
		if spaceID == "" {
			return fmt.Errorf("No space_id is set")
		}

		localeID := rs.Primary.ID
		if localeID == "" {
			return fmt.Errorf("No locale ID is set")
		}

		client := testAccProvider.Meta().(*contentful.Contentful)

		contentfulLocale, err := client.Locales.Get(spaceID, localeID)
		if err != nil {
			return err
		}

		*locale = *contentfulLocale

		return nil
	}
}

func testAccCheckContentfulLocaleAttributes(locale *contentful.Locale, attrs map[string]interface{}) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		name := attrs["name"].(string)
		if locale.Name != name {
			return fmt.Errorf("Locale name does not match: %s, %s", locale.Name, name)
		}

		code := attrs["code"].(string)
		if locale.Code != code {
			return fmt.Errorf("Locale code does not match: %s, %s", locale.Code, code)
		}

		fallbackCode := attrs["fallback_code"].(string)
		if locale.FallbackCode != fallbackCode {
			return fmt.Errorf("Locale fallback code does not match: %s, %s", locale.FallbackCode, fallbackCode)
		}

		isOptional := attrs["optional"].(bool)
		if locale.Optional != isOptional {
			return fmt.Errorf("Locale options value does not match: %t, %t", locale.Optional, isOptional)
		}

		isCDA := attrs["cda"].(bool)
		if locale.CDA != isCDA {
			return fmt.Errorf("Locale cda does not match: %t, %t", locale.CDA, isCDA)
		}

		isCMA := attrs["cma"].(bool)
		if locale.CMA != isCMA {
			return fmt.Errorf("Locale cma does not match: %t, %t", locale.CMA, isCMA)
		}

		return nil
	}
}

func testAccContentfulLocaleDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "contentful_locale" {
			continue
		}

		spaceID := rs.Primary.Attributes["space_id"]
		if spaceID == "" {
			return fmt.Errorf("No space_id is set")
		}

		localeID := rs.Primary.ID
		if localeID == "" {
			return fmt.Errorf("No locale ID is set")
		}

		client := testAccProvider.Meta().(*contentful.Contentful)

		_, err := client.Locales.Get(spaceID, localeID)
		if _, ok := err.(contentful.NotFoundError); ok {
			return nil
		}

		return fmt.Errorf("Locale still exists with id: %s", localeID)
	}

	return nil
}

func testAccContentfulLocaleConfig(spaceName, name string) string {
	return fmt.Sprintf(`
resource "contentful_space" "myspace" {
  name = "%s"
  default_locale = "en-US"
}

resource "contentful_locale" "mylocale" {
  space_id = "${contentful_space.myspace.id}"

  name = "%s"
  code = "de"
  fallback_code = "en-US"
  optional = false
  cda = false
  cma = true
}
`, spaceName, name)
}

func testAccContentfulLocaleUpdateConfig(spaceName, name string) string {
	return fmt.Sprintf(`
resource "contentful_space" "myspace" {
  name = "%s"
  default_locale = "en-US"
}

resource "contentful_locale" "mylocale" {
  space_id = "${contentful_space.myspace.id}"

  name = "%s-updated"
  code = "es"
  fallback_code = "en-US"
  optional = true
  cda = true
  cma = false
}
`, spaceName, name)
}
