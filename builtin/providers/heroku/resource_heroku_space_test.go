package heroku

import (
	"context"
	"fmt"
	"os"
	"testing"

	heroku "github.com/cyberdelia/heroku-go/v3"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccHerokuSpace_Basic(t *testing.T) {
	var space heroku.Space
	spaceName := fmt.Sprintf("tftest-%s", acctest.RandString(10))
	spaceName2 := fmt.Sprintf("tftest-%s", acctest.RandString(10))
	org := os.Getenv("HEROKU_ORGANIZATION")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			if org == "" {
				t.Skip("HEROKU_ORGANIZATION is not set; skipping test.")
			}
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuSpaceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuSpaceConfig_basic(spaceName, org),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuSpaceExists("heroku_space.foobar", &space),
					testAccCheckHerokuSpaceAttributes(&space, spaceName),
				),
			},
			{
				Config: testAccCheckHerokuSpaceConfig_basic(spaceName2, org),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuSpaceExists("heroku_space.foobar", &space),
					testAccCheckHerokuSpaceAttributes(&space, spaceName2),
				),
			},
		},
	})
}

func testAccCheckHerokuSpaceConfig_basic(spaceName, orgName string) string {
	return fmt.Sprintf(`
resource "heroku_space" "foobar" {
  name = "%s"
	organization = "%s"
	region = "virginia"
}
`, spaceName, orgName)
}

func testAccCheckHerokuSpaceExists(n string, space *heroku.Space) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No space name set")
		}

		client := testAccProvider.Meta().(*heroku.Service)

		foundSpace, err := client.SpaceInfo(context.TODO(), rs.Primary.ID)
		if err != nil {
			return err
		}

		if foundSpace.ID != rs.Primary.ID {
			return fmt.Errorf("Space not found")
		}

		*space = *foundSpace

		return nil
	}
}

func testAccCheckHerokuSpaceAttributes(space *heroku.Space, spaceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if space.Name != spaceName {
			return fmt.Errorf("Bad name: %s", space.Name)
		}

		return nil
	}
}

func testAccCheckHerokuSpaceDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*heroku.Service)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "heroku_space" {
			continue
		}

		_, err := client.SpaceInfo(context.TODO(), rs.Primary.ID)

		if err == nil {
			return fmt.Errorf("Space still exists")
		}
	}

	return nil
}
