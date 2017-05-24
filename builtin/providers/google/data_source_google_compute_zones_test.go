package google

import (
	"errors"
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccGoogleComputeZones_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckGoogleComputeZonesConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGoogleComputeZonesMeta("data.google_compute_zones.available"),
				),
			},
		},
	})
}

func testAccCheckGoogleComputeZonesMeta(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Can't find zones data source: %s", n)
		}

		if rs.Primary.ID == "" {
			return errors.New("zones data source ID not set.")
		}

		count, ok := rs.Primary.Attributes["names.#"]
		if !ok {
			return errors.New("can't find 'names' attribute")
		}

		noOfNames, err := strconv.Atoi(count)
		if err != nil {
			return errors.New("failed to read number of zones")
		}
		if noOfNames < 2 {
			return fmt.Errorf("expected at least 2 zones, received %d, this is most likely a bug",
				noOfNames)
		}

		for i := 0; i < noOfNames; i++ {
			idx := "names." + strconv.Itoa(i)
			v, ok := rs.Primary.Attributes[idx]
			if !ok {
				return fmt.Errorf("zone list is corrupt (%q not found), this is definitely a bug", idx)
			}
			if len(v) < 1 {
				return fmt.Errorf("Empty zone name (%q), this is definitely a bug", idx)
			}
		}

		return nil
	}
}

var testAccCheckGoogleComputeZonesConfig = `
data "google_compute_zones" "available" {}
`
