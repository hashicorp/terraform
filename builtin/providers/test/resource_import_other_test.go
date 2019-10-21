package test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestResourceImportOther(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_import_other" "foo" {
}
				`),
			},
			{
				ImportState:  true,
				ResourceName: "test_resource_import_other.foo",

				ImportStateCheck: func(iss []*terraform.InstanceState) error {
					if got, want := len(iss), 2; got != want {
						return fmt.Errorf("wrong number of resources %d; want %d", got, want)
					}

					byID := make(map[string]*terraform.InstanceState, len(iss))
					for _, is := range iss {
						byID[is.ID] = is
					}

					if is, ok := byID["import_other_main"]; !ok {
						return fmt.Errorf("no instance with id import_other_main in state")
					} else if got, want := is.Ephemeral.Type, "test_resource_import_other"; got != want {
						return fmt.Errorf("import_other_main has wrong type %q; want %q", got, want)
					} else if got, want := is.Attributes["computed"], "hello!"; got != want {
						return fmt.Errorf("import_other_main has wrong value %q for its computed attribute; want %q", got, want)
					}
					if is, ok := byID["import_other_other"]; !ok {
						return fmt.Errorf("no instance with id import_other_other in state")
					} else if got, want := is.Ephemeral.Type, "test_resource_defaults"; got != want {
						return fmt.Errorf("import_other_other has wrong type %q; want %q", got, want)
					}

					return nil
				},
			},
		},
	})
}
