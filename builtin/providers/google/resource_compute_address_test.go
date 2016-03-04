package google

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"google.golang.org/api/compute/v1"
)

func TestAccComputeAddress_basic(t *testing.T) {
	var addr compute.Address

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeAddressDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeAddress_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeAddressExists(
						"google_compute_address.foobar", &addr),
				),
			},
		},
	})
}

func testAccCheckComputeAddressDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "google_compute_address" {
			continue
		}

		_, err := config.clientCompute.Addresses.Get(
			config.Project, config.Region, rs.Primary.ID).Do()
		if err == nil {
			return fmt.Errorf("Address still exists")
		}
	}

	return nil
}

func testAccCheckComputeAddressExists(n string, addr *compute.Address) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)

		found, err := config.clientCompute.Addresses.Get(
			config.Project, config.Region, rs.Primary.ID).Do()
		if err != nil {
			return err
		}

		if found.Name != rs.Primary.ID {
			return fmt.Errorf("Addr not found")
		}

		*addr = *found

		return nil
	}
}

var testAccComputeAddress_basic = fmt.Sprintf(`
resource "google_compute_address" "foobar" {
	name = "address-test-%s"
}`, acctest.RandString(10))
