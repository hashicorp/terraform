package test

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestResourceNestedSet_basic(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_nested_set" "foo" {
	single {
		value = "bar"
	}
}
				`),
			},
		},
	})
}

func TestResourceNestedSet_basicImport(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_nested_set" "foo" {
	single {
		value = "bar"
	}
}
				`),
			},
			resource.TestStep{
				ImportState:  true,
				ResourceName: "test_resource_nested_set.foo",
				Config: strings.TrimSpace(`
resource "test_resource_nested_set" "foo" {
	single {
		value = "bar"
	}
}
				`),
				ImportStateCheck: func(ss []*terraform.InstanceState) error {
					for _, s := range ss {
						if s.Attributes["multi.#"] != "0" ||
							s.Attributes["single.#"] != "0" ||
							s.Attributes["type_list.#"] != "0" ||
							s.Attributes["with_list.#"] != "0" {
							return fmt.Errorf("missing blocks in imported state:\n%s", s)
						}
					}
					return nil
				},
			},
		},
	})
}

// The set should not be generated because of it's computed value
func TestResourceNestedSet_noSet(t *testing.T) {
	checkFunc := func(s *terraform.State) error {
		root := s.ModuleByPath(addrs.RootModuleInstance)
		res := root.Resources["test_resource_nested_set.foo"]
		for k, v := range res.Primary.Attributes {
			if strings.HasPrefix(k, "single") && k != "single.#" {
				return fmt.Errorf("unexpected set value: %s:%s", k, v)
			}
		}
		return nil
	}
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_nested_set" "foo" {
}
				`),
				Check: checkFunc,
			},
		},
	})
}

// the empty type_list must be passed to the provider with 1 nil element
func TestResourceNestedSet_emptyBlock(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_nested_set" "foo" {
	type_list {
	}
}
				`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("test_resource_nested_set.foo", "type_list.#", "1"),
				),
			},
		},
	})
}

func TestResourceNestedSet_emptyNestedListBlock(t *testing.T) {
	checkFunc := func(s *terraform.State) error {
		root := s.ModuleByPath(addrs.RootModuleInstance)
		res := root.Resources["test_resource_nested_set.foo"]
		found := false
		for k := range res.Primary.Attributes {
			if !regexp.MustCompile(`^with_list\.\d+\.list_block\.`).MatchString(k) {
				continue
			}
			found = true
		}
		if !found {
			return fmt.Errorf("with_list.X.list_block not found")
		}
		return nil
	}
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_nested_set" "foo" {
	with_list {
		required = "ok"
		list_block {
		}
	}
}
				`),
				Check: checkFunc,
			},
		},
	})
}
func TestResourceNestedSet_emptyNestedList(t *testing.T) {
	checkFunc := func(s *terraform.State) error {
		root := s.ModuleByPath(addrs.RootModuleInstance)
		res := root.Resources["test_resource_nested_set.foo"]
		found := false
		for k, v := range res.Primary.Attributes {
			if regexp.MustCompile(`^with_list\.\d+\.list\.#$`).MatchString(k) {
				found = true
				if v != "0" {
					return fmt.Errorf("expected empty list: %s, got %s", k, v)
				}
				break
			}
		}
		if !found {
			return fmt.Errorf("with_list.X.nested_list not found")
		}
		return nil
	}
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_nested_set" "foo" {
	with_list {
		required = "ok"
		list = []
	}
}
				`),
				Check: checkFunc,
			},
		},
	})
}

func TestResourceNestedSet_addRemove(t *testing.T) {
	var id string
	checkFunc := func(s *terraform.State) error {
		root := s.ModuleByPath(addrs.RootModuleInstance)
		res := root.Resources["test_resource_nested_set.foo"]
		if res.Primary.ID == id {
			return errors.New("expected new resource")
		}
		id = res.Primary.ID
		return nil
	}
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_nested_set" "foo" {
}
				`),
				Check: checkFunc,
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_nested_set" "foo" {
	single {
		value = "bar"
	}
}
				`),
				Check: resource.ComposeTestCheckFunc(
					checkFunc,
					resource.TestCheckResourceAttr(
						"test_resource_nested_set.foo", "single.#", "1",
					),
					// the hash of single seems to change here, so we're not
					// going to test for "value" directly
					// FIXME: figure out why the set hash changes
				),
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_nested_set" "foo" {
}
				`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"test_resource_nested_set.foo", "single.#", "0",
					),
					checkFunc,
				),
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_nested_set" "foo" {
	single {
		value = "bar"
	}
}
				`),
				Check: checkFunc,
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_nested_set" "foo" {
	single {
		value = "bar"
		optional = "baz"
	}
}
				`),
				Check: checkFunc,
			},

			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_nested_set" "foo" {
}
			   	`),
				Check: checkFunc,
			},
		},
	})
}
func TestResourceNestedSet_multiAddRemove(t *testing.T) {
	checkFunc := func(s *terraform.State) error {
		return nil
	}
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_nested_set" "foo" {
}
				`),
				Check: checkFunc,
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_nested_set" "foo" {
	multi {
		optional = "bar"
	}
}
				`),
				Check: checkFunc,
			},

			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_nested_set" "foo" {
}
								`),
				Check: checkFunc,
			},

			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_nested_set" "foo" {
	multi {
		set {
			required = "val"
		}
	}
}
				`),
				Check: checkFunc,
			},

			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_nested_set" "foo" {
	multi {
		set {
			required = "new"
		}
	}
}
				`),
				Check: checkFunc,
			},

			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_nested_set" "foo" {
	multi {
		set {
			required = "new"
			optional_int = 3
		}
	}
}
				`),
				Check: checkFunc,
			},

			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_nested_set" "foo" {
	single {
		value = "bar"
		optional = "baz"
	}
	multi {
		set {
			required = "new"
			optional_int = 3
		}
	}
}
			`),
				Check: checkFunc,
			},

			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_nested_set" "foo" {
	optional = true
	single {
		value = "bar"
		optional = "baz"
	}
	multi {
		set {
			required = "new"
			optional_int = 3
		}
	}
}
			`),
				Check: checkFunc,
			},
		},
	})
}

func TestResourceNestedSet_forceNewEmptyString(t *testing.T) {
	var id string
	step := 0
	checkFunc := func(s *terraform.State) error {
		root := s.ModuleByPath(addrs.RootModuleInstance)
		res := root.Resources["test_resource_nested_set.foo"]
		defer func() {
			step++
			id = res.Primary.ID
		}()

		if step == 2 && res.Primary.ID == id {
			// setting an empty string currently does not trigger ForceNew, but
			// it should in the future.
			return nil
		}

		if res.Primary.ID == id {
			return errors.New("expected new resource")
		}

		return nil
	}
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_nested_set" "foo" {
	multi {
		set {
			required = "val"
		}
	}
}
				`),
				Check: checkFunc,
			},

			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_nested_set" "foo" {
	multi {
		set {
			required = ""
		}
	}
}
				`),
				Check: checkFunc,
			},

			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_nested_set" "foo" {
	force_new = ""
}
				`),
				Check: checkFunc,
			},
		},
	})
}

func TestResourceNestedSet_setWithList(t *testing.T) {
	checkFunc := func(s *terraform.State) error {
		return nil
	}
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_nested_set" "foo" {
	with_list {
		required = "bar"
		list = ["initial value"]
	}
}
				`),
				Check: checkFunc,
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_nested_set" "foo" {
	with_list {
		required = "bar"
		list = ["second value"]
	}
}
				`),
				Check: checkFunc,
			},
		},
	})
}

// This is the same as forceNewEmptyString, but we start with the empty value,
// instead of changing it.
func TestResourceNestedSet_nestedSetEmptyString(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_nested_set" "foo" {
	multi {
		set {
			required = ""
		}
	}
}
				`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"test_resource_nested_set.foo", "multi.529860700.set.4196279896.required", "",
					),
				),
			},
		},
	})
}

func TestResourceNestedSet_emptySet(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_nested_set" "foo" {
	multi {
	}
}
				`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"test_resource_nested_set.foo", "multi.#", "1",
					),
				),
			},
		},
	})
}

func TestResourceNestedSet_multipleUnknownSetElements(t *testing.T) {
	checkFunc := func(s *terraform.State) error {
		return nil
	}
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_nested_set" "a" {
}

resource "test_resource_nested_set" "b" {
}

resource "test_resource_nested_set" "c" {
	multi {
		optional = test_resource_nested_set.a.id
	}
	multi {
		optional = test_resource_nested_set.b.id
	}
}
				`),
				Check: checkFunc,
			},
		},
	})
}

func TestResourceNestedSet_interpolationChanges(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_nested_set" "foo" {
	single {
		value = "x"
	}
}
resource "test_resource_nested_set" "bar" {
	single {
		value = test_resource_nested_set.foo.id
	}
}
				`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"test_resource_nested_set.foo", "single.#", "1",
					),
					resource.TestCheckResourceAttr(
						"test_resource_nested_set.bar", "single.#", "1",
					),
				),
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_nested_set" "baz" {
	single {
		value = "x"
	}
}
resource "test_resource_nested_set" "bar" {
	single {
		value = test_resource_nested_set.baz.id
	}
}
				`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"test_resource_nested_set.baz", "single.#", "1",
					),
					resource.TestCheckResourceAttr(
						"test_resource_nested_set.bar", "single.#", "1",
					),
				),
			},
		},
	})
}

func TestResourceNestedSet_dynamicSetBlock(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource" "a" {
	required = "ok"
	required_map = {
		a = "b"
	}
}

resource "test_resource_nested_set" "foo" {
  dynamic "with_list" {
    iterator = thing
	for_each = test_resource.a.computed_list
    content {
      required = thing.value
	  list = [thing.key]
    }
  }
}
				`),
				Check: resource.ComposeTestCheckFunc(
					func(s *terraform.State) error {
						fmt.Println(s)
						return nil
					},
				),
			},
		},
	})
}
