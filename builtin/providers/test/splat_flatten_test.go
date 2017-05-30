package test

import (
	"errors"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

// This is actually a test of some core functionality in conjunction with
// helper/schema, rather than of the test provider itself.
//
// Here we're just verifying that unknown splats get flattened when assigned
// to list and set attributes. A variety of other situations are tested in
// an apply context test in the core package, but for this part we lean on
// helper/schema and thus need to exercise it at a higher level.

func TestSplatFlatten(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
resource "test_resource" "source" {
	required = "foo ${count.index}"
	required_map = {
	    key = "value"
	}
	count = 3
}

resource "test_resource" "splatted" {
	# This legacy form of splatting into a list is still supported for
	# backward-compatibility but no longer suggested.
	set = ["${test_resource.source.*.computed_from_required}"]
	list = ["${test_resource.source.*.computed_from_required}"]

	required = "yep"
	required_map = {
	    key = "value"
	}
}
				`,
				Check: func(s *terraform.State) error {
					gotAttrs := s.RootModule().Resources["test_resource.splatted"].Primary.Attributes
					t.Logf("attrs %#v", gotAttrs)
					wantAttrs := map[string]string{
						"list.#": "3",
						"list.0": "foo 0",
						"list.1": "foo 1",
						"list.2": "foo 2",

						// This depends on the default set hash implementation.
						// If that changes, these keys will need to be updated.
						"set.#":          "3",
						"set.1136855734": "foo 0",
						"set.885275168":  "foo 1",
						"set.2915920794": "foo 2",
					}
					errored := false
					for k, want := range wantAttrs {
						got := gotAttrs[k]
						if got != want {
							t.Errorf("Wrong %s value %q; want %q", k, got, want)
							errored = true
						}
					}
					if errored {
						return errors.New("incorrect attribute values")
					}
					return nil
				},
			},
		},
	})

}
