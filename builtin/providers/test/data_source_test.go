package test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestDataSource_dataSourceCount(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers: testAccProviders,
		CheckDestroy: func(s *terraform.State) error {
			return nil
		},
		Steps: []resource.TestStep{
			{
				Config: strings.TrimSpace(`
data "test_data_source" "test" {
  count = 3
  input = "count-${count.index}"
}

resource "test_resource" "foo" {
  required     = "yep"
  required_map = {
    key = "value"
  }

  list = ["${data.test_data_source.test.*.output}"]
}
				`),
				Check: func(s *terraform.State) error {
					res, hasRes := s.RootModule().Resources["test_resource.foo"]
					if !hasRes {
						return errors.New("No test_resource.foo in state")
					}
					if res.Primary.Attributes["list.#"] != "3" {
						return errors.New("Wrong list.#, expected 3")
					}
					if res.Primary.Attributes["list.0"] != "count-0" {
						return errors.New("Wrong list.0, expected count-0")
					}
					if res.Primary.Attributes["list.1"] != "count-1" {
						return errors.New("Wrong list.0, expected count-1")
					}
					if res.Primary.Attributes["list.2"] != "count-2" {
						return errors.New("Wrong list.0, expected count-2")
					}
					return nil
				},
			},
		},
	})
}

// Test that the output of a data source can be used as the value for
// a "count" in a real resource. This would fail with "count cannot be computed"
// at some point.
func TestDataSource_valueAsResourceCount(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers: testAccProviders,
		CheckDestroy: func(s *terraform.State) error {
			return nil
		},
		Steps: []resource.TestStep{
			{
				Config: strings.TrimSpace(`
data "test_data_source" "test" {
  input = "4"
}

resource "test_resource" "foo" {
  count = "${data.test_data_source.test.output}"

  required     = "yep"
  required_map = {
    key = "value"
  }
}
				`),
				Check: func(s *terraform.State) error {
					count := 0
					for k, _ := range s.RootModule().Resources {
						if strings.HasPrefix(k, "test_resource.foo.") {
							count++
						}
					}

					if count != 4 {
						return fmt.Errorf("bad count: %d", count)
					}
					return nil
				},
			},
		},
	})
}
