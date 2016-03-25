package terraform

import (
	"reflect"
	"testing"
)

func TestParseResourceAddress(t *testing.T) {
	cases := map[string]struct {
		Input    string
		Expected *ResourceAddress
		Output   string
	}{
		"implicit primary, no specific index": {
			"aws_instance.foo",
			&ResourceAddress{
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        -1,
			},
			"",
		},
		"implicit primary, explicit index": {
			"aws_instance.foo[2]",
			&ResourceAddress{
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        2,
			},
			"",
		},
		"implicit primary, explicit index over ten": {
			"aws_instance.foo[12]",
			&ResourceAddress{
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        12,
			},
			"",
		},
		"explicit primary, explicit index": {
			"aws_instance.foo.primary[2]",
			&ResourceAddress{
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        2,
			},
			"aws_instance.foo[2]",
		},
		"tainted": {
			"aws_instance.foo.tainted",
			&ResourceAddress{
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypeTainted,
				Index:        -1,
			},
			"",
		},
		"deposed": {
			"aws_instance.foo.deposed",
			&ResourceAddress{
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypeDeposed,
				Index:        -1,
			},
			"",
		},
		"with a hyphen": {
			"aws_instance.foo-bar",
			&ResourceAddress{
				Type:         "aws_instance",
				Name:         "foo-bar",
				InstanceType: TypePrimary,
				Index:        -1,
			},
			"",
		},
		"in a module": {
			"module.child.aws_instance.foo",
			&ResourceAddress{
				Path:         []string{"child"},
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        -1,
			},
			"",
		},
		"nested modules": {
			"module.a.module.b.module.forever.aws_instance.foo",
			&ResourceAddress{
				Path:         []string{"a", "b", "forever"},
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        -1,
			},
			"",
		},
		"just a module": {
			"module.a",
			&ResourceAddress{
				Path:         []string{"a"},
				Type:         "",
				Name:         "",
				InstanceType: TypePrimary,
				Index:        -1,
			},
			"",
		},
		"just a nested module": {
			"module.a.module.b",
			&ResourceAddress{
				Path:         []string{"a", "b"},
				Type:         "",
				Name:         "",
				InstanceType: TypePrimary,
				Index:        -1,
			},
			"",
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

		expected := tc.Input
		if tc.Output != "" {
			expected = tc.Output
		}
		if out.String() != expected {
			t.Fatalf("bad: %q\n\nexpected: %s\n\ngot: %s", tn, expected, out)
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
		"index over ten": {
			Address: &ResourceAddress{
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        1,
			},
			Other: &ResourceAddress{
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        13,
			},
			Expect: false,
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
