package test

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestResourceForceNew_create(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_force_new" "foo" {
  triggers = {
	"a" = "foo"
  }
}`),
			},
		},
	})
}
func TestResourceForceNew_update(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_force_new" "foo" {
  triggers = {
	"a" = "foo"
  }
}`),
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_force_new" "foo" {
  triggers = {
	"a" = "bar"
  }
}`),
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_force_new" "foo" {
  triggers = {
	"b" = "bar"
  }
}`),
			},
		},
	})
}

func TestResourceForceNew_remove(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_force_new" "foo" {
  triggers = {
	"a" = "bar"
  }
}`),
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_force_new" "foo" {
}			`),
			},
		},
	})
}
