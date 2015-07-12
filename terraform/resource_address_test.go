package terraform

import (
	"reflect"
	"testing"
)

func TestParseResourceAddress(t *testing.T) {
	cases := map[string]struct {
		Input    string
		Expected *ResourceAddress
	}{
		"implicit primary, no specific index": {
			Input: "aws_instance.foo",
			Expected: &ResourceAddress{
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        -1,
			},
		},
		"implicit primary, explicit index": {
			Input: "aws_instance.foo[2]",
			Expected: &ResourceAddress{
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        2,
			},
		},
		"explicit primary, explicit index": {
			Input: "aws_instance.foo.primary[2]",
			Expected: &ResourceAddress{
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        2,
			},
		},
		"tainted": {
			Input: "aws_instance.foo.tainted",
			Expected: &ResourceAddress{
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypeTainted,
				Index:        -1,
			},
		},
		"deposed": {
			Input: "aws_instance.foo.deposed",
			Expected: &ResourceAddress{
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypeDeposed,
				Index:        -1,
			},
		},
		"with a hyphen": {
			Input: "aws_instance.foo-bar",
			Expected: &ResourceAddress{
				Type:         "aws_instance",
				Name:         "foo-bar",
				InstanceType: TypePrimary,
				Index:        -1,
			},
		},
		"in a module": {
			Input: "module.child.aws_instance.foo",
			Expected: &ResourceAddress{
				Path:         []string{"child"},
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        -1,
			},
		},
		"nested modules": {
			Input: "module.a.module.b.module.forever.aws_instance.foo",
			Expected: &ResourceAddress{
				Path:         []string{"a", "b", "forever"},
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        -1,
			},
		},
		"just a module": {
			Input: "module.a",
			Expected: &ResourceAddress{
				Path:         []string{"a"},
				Type:         "",
				Name:         "",
				InstanceType: TypePrimary,
				Index:        -1,
			},
		},
		"just a nested module": {
			Input: "module.a.module.b",
			Expected: &ResourceAddress{
				Path:         []string{"a", "b"},
				Type:         "",
				Name:         "",
				InstanceType: TypePrimary,
				Index:        -1,
			},
		},
	}

	for tn, tc := range cases {
		out, err := ParseResourceAddress(tc.Input)
		if err != nil {
			t.Fatalf("unexpected err: %#v", err)
		}

		if !reflect.DeepEqual(out, tc.Expected) {
			t.Fatalf("bad: %q\n\nexpected:\n%#v\n\ngot:\n%#v", tn, tc.Expected, out)
		}
	}
}

func TestResourceAddressEquals(t *testing.T) {
	cases := map[string]struct {
		Address *ResourceAddress
		Other   interface{}
		Expect  bool
	}{
		"basic match": {
			Address: &ResourceAddress{
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        0,
			},
			Other: &ResourceAddress{
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        0,
			},
			Expect: true,
		},
		"address does not set index": {
			Address: &ResourceAddress{
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        -1,
			},
			Other: &ResourceAddress{
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        3,
			},
			Expect: true,
		},
		"other does not set index": {
			Address: &ResourceAddress{
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        3,
			},
			Other: &ResourceAddress{
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        -1,
			},
			Expect: true,
		},
		"neither sets index": {
			Address: &ResourceAddress{
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        -1,
			},
			Other: &ResourceAddress{
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        -1,
			},
			Expect: true,
		},
		"different type": {
			Address: &ResourceAddress{
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        0,
			},
			Other: &ResourceAddress{
				Type:         "aws_vpc",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        0,
			},
			Expect: false,
		},
		"different name": {
			Address: &ResourceAddress{
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        0,
			},
			Other: &ResourceAddress{
				Type:         "aws_instance",
				Name:         "bar",
				InstanceType: TypePrimary,
				Index:        0,
			},
			Expect: false,
		},
		"different instance type": {
			Address: &ResourceAddress{
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        0,
			},
			Other: &ResourceAddress{
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypeTainted,
				Index:        0,
			},
			Expect: false,
		},
		"different index": {
			Address: &ResourceAddress{
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        0,
			},
			Other: &ResourceAddress{
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        1,
			},
			Expect: false,
		},
		"module address matches address of resource inside module": {
			Address: &ResourceAddress{
				Path:         []string{"a", "b"},
				Type:         "",
				Name:         "",
				InstanceType: TypePrimary,
				Index:        -1,
			},
			Other: &ResourceAddress{
				Path:         []string{"a", "b"},
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        0,
			},
			Expect: true,
		},
		"module address doesn't match resource outside module": {
			Address: &ResourceAddress{
				Path:         []string{"a", "b"},
				Type:         "",
				Name:         "",
				InstanceType: TypePrimary,
				Index:        -1,
			},
			Other: &ResourceAddress{
				Path:         []string{"a"},
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        0,
			},
			Expect: false,
		},
		"nil path vs empty path should match": {
			Address: &ResourceAddress{
				Path:         []string{},
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        -1,
			},
			Other: &ResourceAddress{
				Path:         nil,
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        0,
			},
			Expect: true,
		},
	}

	for tn, tc := range cases {
		actual := tc.Address.Equals(tc.Other)
		if actual != tc.Expect {
			t.Fatalf("%q: expected equals: %t, got %t for:\n%#v\n%#v",
				tn, tc.Expect, actual, tc.Address, tc.Other)
		}
	}
}
