package test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

// TestResourceDataDep_alignedCountScaleOut tests to make sure interpolation
// works (namely without index errors) when a data source and a resource share
// the same count variable during scale-out with an existing state.
func TestResourceDataDep_alignedCountScaleOut(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers: testAccProviders,
		CheckDestroy: func(s *terraform.State) error {
			return nil
		},
		Steps: []resource.TestStep{
			{
				Config: testResourceDataDepConfig(2),
			},
			{
				Config: testResourceDataDepConfig(4),
				Check:  resource.TestCheckOutput("out", "value_from_api,value_from_api,value_from_api,value_from_api"),
			},
		},
	})
}

// TestResourceDataDep_alignedCountScaleIn tests to make sure interpolation
// works (namely without index errors) when a data source and a resource share
// the same count variable during scale-in with an existing state.
func TestResourceDataDep_alignedCountScaleIn(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers: testAccProviders,
		CheckDestroy: func(s *terraform.State) error {
			return nil
		},
		Steps: []resource.TestStep{
			{
				Config: testResourceDataDepConfig(4),
			},
			{
				Config: testResourceDataDepConfig(2),
				Check:  resource.TestCheckOutput("out", "value_from_api,value_from_api"),
			},
		},
	})
}

// TestDataResourceDep_alignedCountScaleOut functions like
// TestResourceDataDep_alignedCountScaleOut, but with the dependencies swapped
// (resource now depends on data source, a pretty regular use case, but
// included here to check for regressions).
func TestDataResourceDep_alignedCountScaleOut(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers: testAccProviders,
		CheckDestroy: func(s *terraform.State) error {
			return nil
		},
		Steps: []resource.TestStep{
			{
				Config: testDataResourceDepConfig(2),
			},
			{
				Config: testDataResourceDepConfig(4),
				Check:  resource.TestCheckOutput("out", "test,test,test,test"),
			},
		},
	})
}

// TestDataResourceDep_alignedCountScaleIn functions like
// TestResourceDataDep_alignedCountScaleIn, but with the dependencies swapped
// (resource now depends on data source, a pretty regular use case, but
// included here to check for regressions).
func TestDataResourceDep_alignedCountScaleIn(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers: testAccProviders,
		CheckDestroy: func(s *terraform.State) error {
			return nil
		},
		Steps: []resource.TestStep{
			{
				Config: testDataResourceDepConfig(4),
			},
			{
				Config: testDataResourceDepConfig(2),
				Check:  resource.TestCheckOutput("out", "test,test"),
			},
		},
	})
}

// TestResourceResourceDep_alignedCountScaleOut functions like
// TestResourceDataDep_alignedCountScaleOut, but with a resource-to-resource
// dependency instead, a pretty regular use case, but included here to check
// for regressions.
func TestResourceResourceDep_alignedCountScaleOut(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers: testAccProviders,
		CheckDestroy: func(s *terraform.State) error {
			return nil
		},
		Steps: []resource.TestStep{
			{
				Config: testResourceResourceDepConfig(2),
			},
			{
				Config: testResourceResourceDepConfig(4),
				Check:  resource.TestCheckOutput("out", "test,test,test,test"),
			},
		},
	})
}

// TestResourceResourceDep_alignedCountScaleIn functions like
// TestResourceDataDep_alignedCountScaleIn, but with a resource-to-resource
// dependency instead, a pretty regular use case, but included here to check
// for regressions.
func TestResourceResourceDep_alignedCountScaleIn(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers: testAccProviders,
		CheckDestroy: func(s *terraform.State) error {
			return nil
		},
		Steps: []resource.TestStep{
			{
				Config: testResourceResourceDepConfig(4),
			},
			{
				Config: testResourceResourceDepConfig(2),
				Check:  resource.TestCheckOutput("out", "test,test"),
			},
		},
	})
}

func testResourceDataDepConfig(count int) string {
	return fmt.Sprintf(`
variable count {
  default = "%d"
}

resource "test_resource" "foo" {
  count    = "${var.count}"
  required = "yes"

  required_map = {
    "foo" = "bar"
  }
}

data "test_data_source" "bar" {
  count = "${var.count}"
  input = "${test_resource.foo.*.computed_read_only[count.index]}"
}

output "out" {
	value = "${join(",", data.test_data_source.bar.*.output)}"
}
`, count)
}

func testDataResourceDepConfig(count int) string {
	return fmt.Sprintf(`
variable count {
  default = "%d"
}

data "test_data_source" "foo" {
  count = "${var.count}"
  input = "test"
}

resource "test_resource" "bar" {
  count    = "${var.count}"
  required = "yes"
  optional = "${data.test_data_source.foo.*.output[count.index]}"

  required_map = {
    "foo" = "bar"
  }
}

output "out" {
  value = "${join(",", test_resource.bar.*.optional)}"
}
`, count)
}

func testResourceResourceDepConfig(count int) string {
	return fmt.Sprintf(`
variable count {
  default = "%d"
}

resource "test_resource" "foo" {
  count    = "${var.count}"
  required = "yes"
  optional = "test"

  required_map = {
    "foo" = "bar"
  }
}

resource "test_resource" "bar" {
  count    = "${var.count}"
  required = "yes"
  optional = "${test_resource.foo.*.optional[count.index]}"

  required_map = {
    "foo" = "bar"
  }
}

output "out" {
  value = "${join(",", test_resource.bar.*.optional)}"
}
`, count)
}
