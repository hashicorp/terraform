package terraform

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/config"
)

func TestParseResourceAddressInternal(t *testing.T) {
	cases := map[string]struct {
		Input    string
		Expected *ResourceAddress
		Output   string
	}{
		"basic resource": {
			"aws_instance.foo",
			&ResourceAddress{
				Mode:         config.ManagedResourceMode,
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        -1,
			},
			"aws_instance.foo",
		},

		"basic resource with count": {
			"aws_instance.foo.1",
			&ResourceAddress{
				Mode:         config.ManagedResourceMode,
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        1,
			},
			"aws_instance.foo[1]",
		},

		"data resource": {
			"data.aws_ami.foo",
			&ResourceAddress{
				Mode:         config.DataResourceMode,
				Type:         "aws_ami",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        -1,
			},
			"data.aws_ami.foo",
		},

		"data resource with count": {
			"data.aws_ami.foo.1",
			&ResourceAddress{
				Mode:         config.DataResourceMode,
				Type:         "aws_ami",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        1,
			},
			"data.aws_ami.foo[1]",
		},

		"non-data resource with 4 elements": {
			"aws_instance.foo.bar.1",
			nil,
			"",
		},
	}

	for tn, tc := range cases {
		t.Run(tc.Input, func(t *testing.T) {
			out, err := parseResourceAddressInternal(tc.Input)
			if (err != nil) != (tc.Expected == nil) {
				t.Fatalf("%s: unexpected err: %#v", tn, err)
			}
			if err != nil {
				return
			}

			if !reflect.DeepEqual(out, tc.Expected) {
				t.Fatalf("bad: %q\n\nexpected:\n%#v\n\ngot:\n%#v", tn, tc.Expected, out)
			}

			// Compare outputs if those exist
			expected := tc.Input
			if tc.Output != "" {
				expected = tc.Output
			}
			if out.String() != expected {
				t.Fatalf("bad: %q\n\nexpected: %s\n\ngot: %s", tn, expected, out)
			}

			// Compare equality because the internal parse is used
			// to compare equality to equal inputs.
			if !out.Equals(tc.Expected) {
				t.Fatalf("expected equality:\n\n%#v\n\n%#v", out, tc.Expected)
			}
		})
	}
}

func TestParseResourceAddress(t *testing.T) {
	cases := map[string]struct {
		Input    string
		Expected *ResourceAddress
		Output   string
		Err      bool
	}{
		"implicit primary managed instance, no specific index": {
			"aws_instance.foo",
			&ResourceAddress{
				Mode:         config.ManagedResourceMode,
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        -1,
			},
			"",
			false,
		},
		"implicit primary data instance, no specific index": {
			"data.aws_instance.foo",
			&ResourceAddress{
				Mode:         config.DataResourceMode,
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        -1,
			},
			"",
			false,
		},
		"implicit primary, explicit index": {
			"aws_instance.foo[2]",
			&ResourceAddress{
				Mode:         config.ManagedResourceMode,
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        2,
			},
			"",
			false,
		},
		"implicit primary, explicit index over ten": {
			"aws_instance.foo[12]",
			&ResourceAddress{
				Mode:         config.ManagedResourceMode,
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        12,
			},
			"",
			false,
		},
		"explicit primary, explicit index": {
			"aws_instance.foo.primary[2]",
			&ResourceAddress{
				Mode:            config.ManagedResourceMode,
				Type:            "aws_instance",
				Name:            "foo",
				InstanceType:    TypePrimary,
				InstanceTypeSet: true,
				Index:           2,
			},
			"",
			false,
		},
		"tainted": {
			"aws_instance.foo.tainted",
			&ResourceAddress{
				Mode:            config.ManagedResourceMode,
				Type:            "aws_instance",
				Name:            "foo",
				InstanceType:    TypeTainted,
				InstanceTypeSet: true,
				Index:           -1,
			},
			"",
			false,
		},
		"deposed": {
			"aws_instance.foo.deposed",
			&ResourceAddress{
				Mode:            config.ManagedResourceMode,
				Type:            "aws_instance",
				Name:            "foo",
				InstanceType:    TypeDeposed,
				InstanceTypeSet: true,
				Index:           -1,
			},
			"",
			false,
		},
		"with a hyphen": {
			"aws_instance.foo-bar",
			&ResourceAddress{
				Mode:         config.ManagedResourceMode,
				Type:         "aws_instance",
				Name:         "foo-bar",
				InstanceType: TypePrimary,
				Index:        -1,
			},
			"",
			false,
		},
		"managed in a module": {
			"module.child.aws_instance.foo",
			&ResourceAddress{
				Path:         []string{"child"},
				Mode:         config.ManagedResourceMode,
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        -1,
			},
			"",
			false,
		},
		"data in a module": {
			"module.child.data.aws_instance.foo",
			&ResourceAddress{
				Path:         []string{"child"},
				Mode:         config.DataResourceMode,
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        -1,
			},
			"",
			false,
		},
		"nested modules": {
			"module.a.module.b.module.forever.aws_instance.foo",
			&ResourceAddress{
				Path:         []string{"a", "b", "forever"},
				Mode:         config.ManagedResourceMode,
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        -1,
			},
			"",
			false,
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
			false,
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
			false,
		},
		"module missing resource type": {
			"module.name.foo",
			nil,
			"",
			true,
		},
	}

	for tn, tc := range cases {
		t.Run(tn, func(t *testing.T) {
			out, err := ParseResourceAddress(tc.Input)
			if (err != nil) != tc.Err {
				t.Fatalf("%s: unexpected err: %#v", tn, err)
			}
			if tc.Err {
				return
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
		})
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
				Mode:         config.ManagedResourceMode,
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        0,
			},
			Other: &ResourceAddress{
				Mode:         config.ManagedResourceMode,
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        0,
			},
			Expect: true,
		},
		"address does not set index": {
			Address: &ResourceAddress{
				Mode:         config.ManagedResourceMode,
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        -1,
			},
			Other: &ResourceAddress{
				Mode:         config.ManagedResourceMode,
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        3,
			},
			Expect: true,
		},
		"other does not set index": {
			Address: &ResourceAddress{
				Mode:         config.ManagedResourceMode,
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        3,
			},
			Other: &ResourceAddress{
				Mode:         config.ManagedResourceMode,
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        -1,
			},
			Expect: true,
		},
		"neither sets index": {
			Address: &ResourceAddress{
				Mode:         config.ManagedResourceMode,
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        -1,
			},
			Other: &ResourceAddress{
				Mode:         config.ManagedResourceMode,
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        -1,
			},
			Expect: true,
		},
		"index over ten": {
			Address: &ResourceAddress{
				Mode:         config.ManagedResourceMode,
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        1,
			},
			Other: &ResourceAddress{
				Mode:         config.ManagedResourceMode,
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        13,
			},
			Expect: false,
		},
		"different type": {
			Address: &ResourceAddress{
				Mode:         config.ManagedResourceMode,
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        0,
			},
			Other: &ResourceAddress{
				Mode:         config.ManagedResourceMode,
				Type:         "aws_vpc",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        0,
			},
			Expect: false,
		},
		"different mode": {
			Address: &ResourceAddress{
				Mode:         config.ManagedResourceMode,
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        0,
			},
			Other: &ResourceAddress{
				Mode:         config.DataResourceMode,
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        0,
			},
			Expect: false,
		},
		"different name": {
			Address: &ResourceAddress{
				Mode:         config.ManagedResourceMode,
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        0,
			},
			Other: &ResourceAddress{
				Mode:         config.ManagedResourceMode,
				Type:         "aws_instance",
				Name:         "bar",
				InstanceType: TypePrimary,
				Index:        0,
			},
			Expect: false,
		},
		"different instance type": {
			Address: &ResourceAddress{
				Mode:         config.ManagedResourceMode,
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        0,
			},
			Other: &ResourceAddress{
				Mode:         config.ManagedResourceMode,
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypeTainted,
				Index:        0,
			},
			Expect: false,
		},
		"different index": {
			Address: &ResourceAddress{
				Mode:         config.ManagedResourceMode,
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        0,
			},
			Other: &ResourceAddress{
				Mode:         config.ManagedResourceMode,
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        1,
			},
			Expect: false,
		},
		"module address matches address of managed resource inside module": {
			Address: &ResourceAddress{
				Path:         []string{"a", "b"},
				Type:         "",
				Name:         "",
				InstanceType: TypePrimary,
				Index:        -1,
			},
			Other: &ResourceAddress{
				Path:         []string{"a", "b"},
				Mode:         config.ManagedResourceMode,
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        0,
			},
			Expect: true,
		},
		"module address matches address of data resource inside module": {
			Address: &ResourceAddress{
				Path:         []string{"a", "b"},
				Type:         "",
				Name:         "",
				InstanceType: TypePrimary,
				Index:        -1,
			},
			Other: &ResourceAddress{
				Path:         []string{"a", "b"},
				Mode:         config.DataResourceMode,
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        0,
			},
			Expect: true,
		},
		"module address doesn't match managed resource outside module": {
			Address: &ResourceAddress{
				Path:         []string{"a", "b"},
				Type:         "",
				Name:         "",
				InstanceType: TypePrimary,
				Index:        -1,
			},
			Other: &ResourceAddress{
				Path:         []string{"a"},
				Mode:         config.ManagedResourceMode,
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        0,
			},
			Expect: false,
		},
		"module address doesn't match data resource outside module": {
			Address: &ResourceAddress{
				Path:         []string{"a", "b"},
				Type:         "",
				Name:         "",
				InstanceType: TypePrimary,
				Index:        -1,
			},
			Other: &ResourceAddress{
				Path:         []string{"a"},
				Mode:         config.DataResourceMode,
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
				Mode:         config.ManagedResourceMode,
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        -1,
			},
			Other: &ResourceAddress{
				Path:         nil,
				Mode:         config.ManagedResourceMode,
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

func TestResourceAddressStateId(t *testing.T) {
	cases := map[string]struct {
		Input    *ResourceAddress
		Expected string
	}{
		"basic resource": {
			&ResourceAddress{
				Mode:         config.ManagedResourceMode,
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        -1,
			},
			"aws_instance.foo",
		},

		"basic resource with index": {
			&ResourceAddress{
				Mode:         config.ManagedResourceMode,
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        2,
			},
			"aws_instance.foo.2",
		},

		"data resource": {
			&ResourceAddress{
				Mode:         config.DataResourceMode,
				Type:         "aws_instance",
				Name:         "foo",
				InstanceType: TypePrimary,
				Index:        -1,
			},
			"data.aws_instance.foo",
		},
	}

	for tn, tc := range cases {
		t.Run(tn, func(t *testing.T) {
			actual := tc.Input.stateId()
			if actual != tc.Expected {
				t.Fatalf("bad: %q\n\nexpected: %s\n\ngot: %s", tn, tc.Expected, actual)
			}
		})
	}
}
