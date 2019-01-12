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
	checkFunc := func(s *terraform.State) error {
		root := s.ModuleByPath(addrs.RootModuleInstance)
		res := root.Resources["test_resource_nested_set.foo"]
		for k, v := range res.Primary.Attributes {
			if strings.HasPrefix(k, "type_list") && v != "1" {
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
	type_list {
	}
}
				`),
				Check: checkFunc,
			},
		},
	})
}

func TestResourceNestedSet_emptyNestedListBlock(t *testing.T) {
	checkFunc := func(s *terraform.State) error {
		root := s.ModuleByPath(addrs.RootModuleInstance)
		res := root.Resources["test_resource_nested_set.foo"]
		found := false
		for k, v := range res.Primary.Attributes {
			if !regexp.MustCompile(`^with_list\.\d+\.list_block\.`).MatchString(k) {
				continue
			}
			found = true

			if strings.HasSuffix(k, ".#") {
				if v != "1" {
					return fmt.Errorf("expected block with no objects: got %s:%s", k, v)
				}
				continue
			}

			// there should be no other attribute values for an empty block
			return fmt.Errorf("unexpected attribute: %s:%s", k, v)
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
	multi {
		set {
			required = ""
		}
	}
}
				`),
				Check: checkFunc,
			},
		},
	})
}

func TestResourceNestedSet_emptySet(t *testing.T) {
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
	multi {
	}
}
				`),
				Check: checkFunc,
			},
		},
	})
}
