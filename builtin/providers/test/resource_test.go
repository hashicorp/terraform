package test

import (
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func TestResource_basic(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource" "foo" {
	required = "yep"
	required_map = {
	    key = "value"
	}
}
				`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr(
						"test_resource.foo", "list.#",
					),
				),
			},
		},
	})
}

func TestResource_changedList(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			{
				Config: strings.TrimSpace(`
resource "test_resource" "foo" {
	required = "yep"
	required_map = {
	    key = "value"
	}
}
				`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr(
						"test_resource.foo", "list.#",
					),
				),
			},
			{
				Config: strings.TrimSpace(`
resource "test_resource" "foo" {
	required = "yep"
	required_map = {
	    key = "value"
	}
	list = ["a"]
}
				`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"test_resource.foo", "list.#", "1",
					),
					resource.TestCheckResourceAttr(
						"test_resource.foo", "list.0", "a",
					),
				),
			},
			{
				Config: strings.TrimSpace(`
resource "test_resource" "foo" {
	required = "yep"
	required_map = {
	    key = "value"
	}
	list = ["a", "b"]
}
				`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"test_resource.foo", "list.#", "2",
					),
					resource.TestCheckResourceAttr(
						"test_resource.foo", "list.0", "a",
					),
					resource.TestCheckResourceAttr(
						"test_resource.foo", "list.1", "b",
					),
				),
			},
			{
				Config: strings.TrimSpace(`
resource "test_resource" "foo" {
	required = "yep"
	required_map = {
	    key = "value"
	}
	list = ["b"]
}
				`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"test_resource.foo", "list.#", "1",
					),
					resource.TestCheckResourceAttr(
						"test_resource.foo", "list.0", "b",
					),
				),
			},
		},
	})
}

// Targeted test in TestContext2Apply_ignoreChangesCreate
func TestResource_ignoreChangesRequired(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource" "foo" {
        required = "yep"
	required_map = {
	    key = "value"
	}
        lifecycle {
                ignore_changes = ["required"]
        }
}
                               `),
				Check: func(s *terraform.State) error {
					return nil
				},
			},
		},
	})
}

func TestResource_ignoreChangesEmpty(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource" "foo" {
	required = "yep"
	required_map = {
	    key = "value"
	}
	optional_force_new = "one"
	lifecycle {
		ignore_changes = []
	}
}
				`),
				Check: func(s *terraform.State) error {
					return nil
				},
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource" "foo" {
	required = "yep"
	required_map = {
	    key = "value"
	}
	optional_force_new = "two"
	lifecycle {
		ignore_changes = []
	}
}
				`),
				Check: func(s *terraform.State) error {
					return nil
				},
			},
		},
	})
}

func TestResource_ignoreChangesForceNew(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource" "foo" {
	required           = "yep"
	required_map = {
	    key = "value"
	}
	optional_force_new = "one"
	lifecycle {
		ignore_changes = ["optional_force_new"]
	}
}
				`),
				Check: func(s *terraform.State) error {
					return nil
				},
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource" "foo" {
	required           = "yep"
	required_map = {
	    key = "value"
	}
	optional_force_new = "two"
	lifecycle {
		ignore_changes = ["optional_force_new"]
	}
}
				`),
				Check: func(s *terraform.State) error {
					return nil
				},
			},
		},
	})
}

// Covers specific scenario in #6005, handled by normalizing boolean strings in
// helper/schema
func TestResource_ignoreChangesForceNewBoolean(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource" "foo" {
  required           = "yep"
  required_map = {
    key = "value"
  }
  optional_force_new = "one"
  optional_bool      = true
  lifecycle {
    ignore_changes = ["optional_force_new"]
  }
}
				`),
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource" "foo" {
  required           = "yep"
  required_map = {
    key = "value"
  }
  optional_force_new = "two"
  optional_bool      = true
  lifecycle {
    ignore_changes = ["optional_force_new"]
  }
}
				`),
				Check: func(s *terraform.State) error {
					return nil
				},
			},
		},
	})
}

func TestResource_ignoreChangesMap(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource" "foo" {
	required           = "yep"
	required_map = {
	  key = "value"
	}
	optional_computed_map = {
		foo = "bar"
	}
	lifecycle {
		ignore_changes = ["optional_computed_map"]
	}
}
				`),
				Check: func(s *terraform.State) error {
					return nil
				},
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource" "foo" {
	required           = "yep"
	required_map = {
	  key = "value"
	}
	optional_computed_map = {
		foo = "bar"
		no  = "update"
	}
	lifecycle {
		ignore_changes = ["optional_computed_map"]
	}
}
				`),
				Check: func(s *terraform.State) error {
					return nil
				},
			},
		},
	})
}

func TestResource_ignoreChangesDependent(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource" "foo" {
	count = 2
	required = "yep"
	required_map = {
		key = "value"
	}

	optional_force_new = "one"
	lifecycle {
		ignore_changes = ["optional_force_new"]
	}
}
resource "test_resource" "bar" {
	count = 2
	required = "yep"
	required_map = {
		key = "value"
	}
	optional = "${element(test_resource.foo.*.id, count.index)}"
}
				`),
				Check: func(s *terraform.State) error {
					return nil
				},
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource" "foo" {
	count = 2
	required = "yep"
	required_map = {
		key = "value"
	}

	optional_force_new = "two"
	lifecycle {
		ignore_changes = ["optional_force_new"]
	}
}
resource "test_resource" "bar" {
	count = 2
	required = "yep"
	required_map = {
		key = "value"
	}
	optional = "${element(test_resource.foo.*.id, count.index)}"
}
				`),
				Check: func(s *terraform.State) error {
					return nil
				},
			},
		},
	})
}

func TestResource_ignoreChangesStillReplaced(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource" "foo" {
  required     = "yep"
  required_map = {
    key = "value"
  }
  optional_force_new = "one"
  optional_bool      = true
  lifecycle {
    ignore_changes = ["optional_bool"]
  }
}
				`),
				Check: func(s *terraform.State) error {
					return nil
				},
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource" "foo" {
  required     = "yep"
  required_map = {
    key = "value"
  }
  optional_force_new = "two"
  optional_bool      = false
  lifecycle {
    ignore_changes = ["optional_bool"]
  }
}
				`),
				Check: func(s *terraform.State) error {
					return nil
				},
			},
		},
	})
}

// Reproduces plan-time panic when the wrong type is interpolated in a list of
// maps.
// TODO: this should return a type error, rather than silently setting an empty
//       list
func TestResource_dataSourceListMapPanic(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource" "foo" {
  required = "val"
  required_map = {x = "y"}
  list_of_map = "${var.maplist}"
}

variable "maplist" {
  type = "list"

  default = [
    {a = "b"}
  ]
}
				`),
				ExpectError: nil,
				Check: func(s *terraform.State) error {
					return nil
				},
			},
		},
	})
}

func TestResource_dataSourceIndexMapList(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource" "foo" {
  required = "val"

  required_map = {
    x = "y"
  }

  list_of_map = [
    {
      a = "1"
      b = "2"
    },
    {
      c = "3"
      d = "4"
    },
  ]
}

output "map_from_list" {
  value = "${test_resource.foo.list_of_map[0]}"
}

output "value_from_map_from_list" {
  value = "${lookup(test_resource.foo.list_of_map[1], "d")}"
}
				`),
				ExpectError: nil,
				Check: func(s *terraform.State) error {
					root := s.ModuleByPath(addrs.RootModuleInstance)
					mapOut := root.Outputs["map_from_list"].Value
					expectedMapOut := map[string]interface{}{
						"a": "1",
						"b": "2",
					}

					valueOut := root.Outputs["value_from_map_from_list"].Value
					expectedValueOut := "4"

					if !reflect.DeepEqual(mapOut, expectedMapOut) {
						t.Fatalf("Expected: %#v\nGot: %#v", expectedMapOut, mapOut)
					}
					if !reflect.DeepEqual(valueOut, expectedValueOut) {
						t.Fatalf("Expected: %#v\nGot: %#v", valueOut, expectedValueOut)
					}
					return nil
				},
			},
		},
	})
}

func testAccCheckResourceDestroy(s *terraform.State) error {
	return nil
}

func TestResource_removeForceNew(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource" "foo" {
	required           = "yep"
	required_map = {
	  key = "value"
	}
	optional_force_new = "here"
}
				`),
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource" "foo" {
	required           = "yep"
	required_map = {
	  key = "value"
	}
}
				`),
			},
		},
	})
}

func TestResource_unknownFuncInMap(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource" "foo" {
	required           = "ok"
	required_map = {
	  key = "${uuid()}"
	}
}
				`),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// Verify that we can destroy when a managed resource references something with
// a count of 1.
func TestResource_countRefDestroyError(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: strings.TrimSpace(`
resource "test_resource" "one" {
	count = 1
	required     = "ok"
	required_map = {
	  key = "val"
	}
}

resource "test_resource" "two" {
	required     = test_resource.one[0].id
	required_map = {
	  key = "val"
	}
}
				`),
			},
		},
	})
}

func TestResource_emptyMapValue(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource" "foo" {
	required     = "ok"
	required_map = {
		a = "a"
		b = ""
	}
}
				`),
			},
		},
	})
}

func TestResource_updateError(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource" "foo" {
  required     = "first"
  required_map = {
    a = "a"
  }
}
`),
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource" "foo" {
  required     = "second"
  required_map = {
    a = "a"
  }
  apply_error = "update_error"
}
`),
				ExpectError: regexp.MustCompile("update_error"),
			},
		},
	})
}

func TestResource_applyError(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource" "foo" {
  required     = "second"
  required_map = {
    a = "a"
  }
  apply_error = "apply_error"
}
`),
				ExpectError: regexp.MustCompile("apply_error"),
			},
		},
	})
}

func TestResource_emptyStrings(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource" "foo" {
  required     = "second"
  required_map = {
    a = "a"
  }

  list = [""]
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("test_resource.foo", "list.0", ""),
				),
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource" "foo" {
  required     = "second"
  required_map = {
    a = "a"
  }

  list = ["", "b"]
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("test_resource.foo", "list.0", ""),
					resource.TestCheckResourceAttr("test_resource.foo", "list.1", "b"),
				),
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource" "foo" {
  required     = "second"
  required_map = {
    a = "a"
  }

  list = [""]
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("test_resource.foo", "list.0", ""),
				),
			},
		},
	})
}

func TestResource_setDrift(t *testing.T) {
	testProvider := testAccProviders["test"]
	res := testProvider.(*schema.Provider).ResourcesMap["test_resource"]

	// reset the Read function after the test
	defer func() {
		res.Read = testResourceRead
	}()

	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource" "foo" {
  required     = "first"
  required_map = {
    a = "a"
	}
	set = ["a", "b"]
}
`),
				Check: func(s *terraform.State) error {
					return nil
				},
			},
			resource.TestStep{
				PreConfig: func() {
					// update the Read function to return the wrong "set" attribute values.
					res.Read = func(d *schema.ResourceData, meta interface{}) error {
						// update as expected first
						if err := testResourceRead(d, meta); err != nil {
							return err
						}
						d.Set("set", []interface{}{"a", "x"})
						return nil
					}
				},
				// Leave the config, so we can detect the mismatched set values.
				// Updating the config would force the test to pass even if the Read
				// function values were ignored.
				Config: strings.TrimSpace(`
resource "test_resource" "foo" {
  required     = "second"
  required_map = {
    a = "a"
	}
	set = ["a", "b"]
}
`),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestResource_optionalComputedMap(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource" "foo" {
	required           = "yep"
	required_map = {
	  key = "value"
	}
	optional_computed_map = {
		foo = "bar"
		baz = ""
	}
}
				`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"test_resource.foo", "optional_computed_map.foo", "bar",
					),
					resource.TestCheckResourceAttr(
						"test_resource.foo", "optional_computed_map.baz", "",
					),
				),
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource" "foo" {
	required           = "yep"
	required_map = {
	  key = "value"
	}
	optional_computed_map = {}
}
				`),
				// removing the map from the config should still leave an empty computed map
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"test_resource.foo", "optional_computed_map.%", "0",
					),
				),
			},
		},
	})
}

func TestResource_plannedComputed(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource" "foo" {
	required = "ok"
	required_map = {
	  key = "value"
	}
	optional = "hi"
}
				`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"test_resource.foo", "planned_computed", "hi",
					),
				),
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource" "foo" {
	required = "ok"
	required_map = {
	  key = "value"
	}
	optional = "changed"
}
				`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"test_resource.foo", "planned_computed", "changed",
					),
				),
			},
		},
	})
}

func TestDiffApply_map(t *testing.T) {
	resSchema := map[string]*schema.Schema{
		"map": {
			Type:     schema.TypeMap,
			Optional: true,
			Computed: true,
			Elem:     &schema.Schema{Type: schema.TypeString},
		},
	}

	priorAttrs := map[string]string{
		"id":      "ok",
		"map.%":   "2",
		"map.foo": "bar",
		"map.bar": "",
	}

	diff := &terraform.InstanceDiff{
		Attributes: map[string]*terraform.ResourceAttrDiff{
			"map.foo": &terraform.ResourceAttrDiff{Old: "bar", New: "", NewRemoved: true},
			"map.bar": &terraform.ResourceAttrDiff{Old: "", New: "", NewRemoved: true},
		},
	}

	newAttrs, err := diff.Apply(priorAttrs, (&schema.Resource{Schema: resSchema}).CoreConfigSchema())
	if err != nil {
		t.Fatal(err)
	}

	expect := map[string]string{
		"id":    "ok",
		"map.%": "0",
	}

	if !reflect.DeepEqual(newAttrs, expect) {
		t.Fatalf("expected:%#v got:%#v", expect, newAttrs)
	}
}

func TestResource_dependsComputed(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
variable "change" {
	default = false
}

resource "test_resource" "foo" {
	required = "ok"
	required_map = {
	    key = "value"
	}
	optional = var.change ? "after" : ""
}

resource "test_resource" "bar" {
	count = var.change ? 1 : 0
	required = test_resource.foo.planned_computed
	required_map = {
		key = "value"
	}
}
				`),
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
variable "change" {
	default = true
}

resource "test_resource" "foo" {
	required = "ok"
	required_map = {
	    key = "value"
	}
	optional = var.change ? "after" : ""
}

resource "test_resource" "bar" {
	count = var.change ? 1 : 0
	required = test_resource.foo.planned_computed
	required_map = {
		key = "value"
	}
}
				`),
			},
		},
	})
}

func TestResource_optionalComputedBool(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource" "foo" {
	required = "yep"
	required_map = {
	    key = "value"
	}
}
				`),
			},
		},
	})
}

func TestResource_replacedOptionalComputed(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_nested" "a" {
}

resource "test_resource" "foo" {
	required = "yep"
	required_map = {
	    key = "value"
	}
	optional_computed = test_resource_nested.a.id
}
				`),
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_nested" "b" {
}

resource "test_resource" "foo" {
	required = "yep"
	required_map = {
	    key = "value"
	}
	optional_computed = test_resource_nested.b.id
}
				`),
			},
		},
	})
}
