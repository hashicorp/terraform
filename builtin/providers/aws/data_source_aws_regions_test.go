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

func TestAccAWSRegions_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: `data "aws_regions" "test" {}`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsRegionsMeta("data.aws_regions.test"),
				),
			},
		},
	})
}

func testAccCheckAwsRegionsMeta(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Can't find regions resource: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("Resource ID not set.")
		}

		actual, err := testAccCheckAwsAvailabilityZonesBuildAvailable(rs.Primary.Attributes)
		if err != nil {
			return err
		}

		expected := actual
		sort.Strings(expected)
		if reflect.DeepEqual(expected, actual) != true {
			return fmt.Errorf("Regions not sorted - expected %v, got %v", expected, actual)
		}

		return nil
	}
}

func testAccCheckAwsRegionsBuild(attrs map[string]string) ([]string, error) {
	v, ok := attrs["names.#"]
	if !ok {
		return nil, fmt.Errorf("Region list is missing.")
	}

	qty, err := strconv.Atoi(v)
	if err != nil {
		return nil, err
	}

	if qty < 1 {
		return nil, fmt.Errorf("No regions found, this is probably a bug.")
	}

	regions := make([]string, qty)
	for n := range regions {
		region, ok := attrs["names."+strconv.Itoa(n)]
		if !ok {
			return nil, fmt.Errorf("Region list corrupt, this is definitely a bug.")
		}
		regions[n] = region
	}

	return regions, nil
}
