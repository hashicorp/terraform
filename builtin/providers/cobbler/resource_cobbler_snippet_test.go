package cobbler

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	cobbler "github.com/jtopjian/cobblerclient"
)

func TestAccCobblerSnippet_basic(t *testing.T) {
	var snippet cobbler.Snippet

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccCobblerPreCheck(t) },
		Providers:    testAccCobblerProviders,
		CheckDestroy: testAccCobblerCheckSnippetDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCobblerSnippet_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCobblerCheckSnippetExists(t, "cobbler_snippet.foo", &snippet),
				),
			},
		},
	})
}

func testAccCobblerCheckSnippetDestroy(s *terraform.State) error {
	config := testAccCobblerProvider.Meta().(*Config)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "cobbler_snippet" {
			continue
		}

		if _, err := config.cobblerClient.GetSnippet(rs.Primary.ID); err == nil {
			return fmt.Errorf("Snippet still exists")
		}
	}

	return nil
}

func testAccCobblerCheckSnippetExists(t *testing.T, n string, snippet *cobbler.Snippet) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccCobblerProvider.Meta().(*Config)

		found, err := config.cobblerClient.GetSnippet(rs.Primary.ID)
		if err != nil {
			return err
		}

		if found.Name != rs.Primary.ID {
			return fmt.Errorf("Snippet not found")
		}

		*snippet = *found

		return nil
	}
}

var testAccCobblerSnippet_basic = `
	resource "cobbler_snippet" "foo" {
		name = "/var/lib/cobbler/snippets/foo"
		body = "I'm a Snippet."
	}`
