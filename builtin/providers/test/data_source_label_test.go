package test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestProviderLabelDataSource(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers: testAccProviders,
		CheckDestroy: func(s *terraform.State) error {
			return nil
		},
		Steps: []resource.TestStep{
			{
				Config: strings.TrimSpace(`
provider "test" {
  label = "foo"
}

data "test_provider_label" "test" {
}
				`),
				Check: func(s *terraform.State) error {
					res, hasRes := s.RootModule().Resources["data.test_provider_label.test"]
					if !hasRes {
						return errors.New("No test_provider_label in state")
					}
					if got, want := res.Primary.ID, "foo"; got != want {
						return fmt.Errorf("wrong id %q; want %q", got, want)
					}
					if got, want := res.Primary.Attributes["label"], "foo"; got != want {
						return fmt.Errorf("wrong id %q; want %q", got, want)
					}
					return nil
				},
			},
		},
	})
}
