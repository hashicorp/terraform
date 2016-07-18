package aws

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSAvailabilityZones_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckAwsAvailabilityZonesConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsAvailabilityZonesMeta("data.aws_availability_zones.availability_zones"),
				),
			},
		},
	})
}

func testAccCheckAwsAvailabilityZonesMeta(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Can't find AZ resource: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("AZ resource ID not set")
		}

		actual, err := testAccCheckAwsAvailabilityZonesBuildAvailable(rs.Primary.Attributes)
		if err != nil {
			return err
		}

		expected := actual
		sort.Strings(expected)
		if reflect.DeepEqual(expected, actual) != true {
			return fmt.Errorf("AZs not sorted - expected %v, got %v", expected, actual)
		}
		return nil
	}
}

func testAccCheckAwsAvailabilityZonesBuildAvailable(attrs map[string]string) ([]string, error) {
	v, ok := attrs["names.#"]
	if !ok {
		return nil, fmt.Errorf("Available AZ list is missing")
	}
	qty, err := strconv.Atoi(v)
	if err != nil {
		return nil, err
	}
	if qty < 1 {
		return nil, fmt.Errorf("No AZs found in region, this is probably a bug.")
	}
	zones := make([]string, qty)
	for n := range zones {
		zone, ok := attrs["names."+strconv.Itoa(n)]
		if !ok {
			return nil, fmt.Errorf("AZ list corrupt, this is definitely a bug")
		}
		zones[n] = zone
	}
	return zones, nil
}

const testAccCheckAwsAvailabilityZonesConfig = `
data "aws_availability_zones" "availability_zones" {
}
`
