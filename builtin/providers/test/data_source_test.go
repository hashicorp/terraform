package test

import (
	"errors"
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
