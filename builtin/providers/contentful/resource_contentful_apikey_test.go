package contentful

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	contentful "github.com/contentful-labs/contentful-go"
)

func TestAccContentfulAPIKey_Basic(t *testing.T) {
	var apiKey contentful.APIKey

	spaceName := fmt.Sprintf("space-name-%s", acctest.RandString(3))
	name := fmt.Sprintf("apikey-name-%s", acctest.RandString(3))
	description := fmt.Sprintf("apikey-description-%s", acctest.RandString(3))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccContentfulAPIKeyDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccContentfulAPIKeyConfig(spaceName, name, description),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckContentfulAPIKeyExists("contentful_apikey.myapikey", &apiKey),
					testAccCheckContentfulAPIKeyAttributes(&apiKey, map[string]interface{}{
						"name":        name,
						"description": description,
					}),
				),
			},
			resource.TestStep{
				Config: testAccContentfulAPIKeyUpdateConfig(spaceName, name, description),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckContentfulAPIKeyExists("contentful_apikey.myapikey", &apiKey),
					testAccCheckContentfulAPIKeyAttributes(&apiKey, map[string]interface{}{
						"name":        fmt.Sprintf("%s-updated", name),
						"description": fmt.Sprintf("%s-updated", description),
					}),
				),
			},
		},
	})
}

func testAccCheckContentfulAPIKeyExists(n string, apiKey *contentful.APIKey) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not Found: %s", n)
		}

		spaceID := rs.Primary.Attributes["space_id"]
		if spaceID == "" {
			return fmt.Errorf("No space_id is set")
		}

		apiKeyID := rs.Primary.ID
		if apiKeyID == "" {
			return fmt.Errorf("No api key ID is set")
		}

		client := testAccProvider.Meta().(*contentful.Contentful)

		contentfulAPIKey, err := client.APIKeys.Get(spaceID, apiKeyID)
		if err != nil {
			return err
		}

		*apiKey = *contentfulAPIKey

		return nil
	}
}

func testAccCheckContentfulAPIKeyAttributes(apiKey *contentful.APIKey, attrs map[string]interface{}) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		name := attrs["name"].(string)
		if apiKey.Name != name {
			return fmt.Errorf("APIKey name does not match: %s, %s", apiKey.Name, name)
		}

		description := attrs["description"].(string)
		if apiKey.Description != description {
			return fmt.Errorf("APIKey description does not match: %s, %s", apiKey.Description, description)
		}

		return nil
	}
}

func testAccContentfulAPIKeyDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "contentful_apikey" {
			continue
		}

		// get space id from resource data
		spaceID := rs.Primary.Attributes["space_id"]
		if spaceID == "" {
			return fmt.Errorf("No space_id is set")
		}

		apiKeyID := rs.Primary.ID
		if apiKeyID == "" {
			return fmt.Errorf("No apikey ID is set")
		}

		client := testAccProvider.Meta().(*contentful.Contentful)

		_, err := client.APIKeys.Get(spaceID, apiKeyID)
		if _, ok := err.(contentful.NotFoundError); ok {
			return nil
		}

		return fmt.Errorf("Api Key still exists with id: %s", rs.Primary.ID)
	}

	return nil
}

func testAccContentfulAPIKeyConfig(spaceName, name, description string) string {
	return fmt.Sprintf(`
resource "contentful_space" "myspace" {
  name = "%s"
}

resource "contentful_apikey" "myapikey" {
  space_id = "${contentful_space.myspace.id}"

  name = "%s"
  description = "%s"
}
`, spaceName, name, description)
}

func testAccContentfulAPIKeyUpdateConfig(spaceName, name, description string) string {
	return fmt.Sprintf(`
resource "contentful_space" "myspace" {
  name = "%s"
}

resource "contentful_apikey" "myapikey" {
  space_id = "${contentful_space.myspace.id}"

  name = "%s-updated"
  description = "%s-updated"
}
`, spaceName, name, description)
}
