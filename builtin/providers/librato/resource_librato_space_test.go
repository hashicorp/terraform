package librato

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/henrikhodne/go-librato/librato"
)

func TestAccLibratoSpace_Basic(t *testing.T) {
	var space librato.Space
	name := acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLibratoSpaceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckLibratoSpaceConfig_basic(name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckLibratoSpaceExists("librato_space.foobar", &space),
					testAccCheckLibratoSpaceAttributes(&space, name),
					resource.TestCheckResourceAttr(
						"librato_space.foobar", "name", name),
				),
			},
		},
	})
}

func testAccCheckLibratoSpaceDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*librato.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "librato_space" {
			continue
		}

		id, err := strconv.ParseUint(rs.Primary.ID, 10, 0)
		if err != nil {
			return fmt.Errorf("ID not a number")
		}

		_, _, err = client.Spaces.Get(uint(id))

		if err == nil {
			return fmt.Errorf("Space still exists")
		}
	}

	return nil
}

func testAccCheckLibratoSpaceAttributes(space *librato.Space, name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if space.Name == nil || *space.Name != name {
			return fmt.Errorf("Bad name: %s", *space.Name)
		}

		return nil
	}
}

func testAccCheckLibratoSpaceExists(n string, space *librato.Space) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Space ID is set")
		}

		client := testAccProvider.Meta().(*librato.Client)

		id, err := strconv.ParseUint(rs.Primary.ID, 10, 0)
		if err != nil {
			return fmt.Errorf("ID not a number")
		}

		foundSpace, _, err := client.Spaces.Get(uint(id))

		if err != nil {
			return err
		}

		if foundSpace.ID == nil || *foundSpace.ID != uint(id) {
			return fmt.Errorf("Space not found")
		}

		*space = *foundSpace

		return nil
	}
}

func testAccCheckLibratoSpaceConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "librato_space" "foobar" {
    name = "%s"
}`, name)
}
