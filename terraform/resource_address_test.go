package terraform

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/addrs"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/configs"
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

func TestResourceAddressContains(t *testing.T) {
	tests := []struct {
		Address *ResourceAddress
		Other   *ResourceAddress
		Want    bool
	}{
		{
			&ResourceAddress{
				Mode:            config.ManagedResourceMode,
				Type:            "aws_instance",
				Name:            "foo",
				InstanceTypeSet: true,
				InstanceType:    TypePrimary,
				Index:           0,
			},
			&ResourceAddress{
				Mode:            config.ManagedResourceMode,
				Type:            "aws_instance",
				Name:            "foo",
				InstanceTypeSet: true,
				InstanceType:    TypePrimary,
				Index:           0,
			},
			true,
		},
		{
			&ResourceAddress{
				Mode:            config.ManagedResourceMode,
				Type:            "aws_instance",
				Name:            "foo",
				InstanceTypeSet: false,
				Index:           0,
			},
			&ResourceAddress{
				Mode:            config.ManagedResourceMode,
				Type:            "aws_instance",
				Name:            "foo",
				InstanceTypeSet: true,
				InstanceType:    TypePrimary,
				Index:           0,
			},
			true,
		},
		{
			&ResourceAddress{
				Mode:            config.ManagedResourceMode,
				Type:            "aws_instance",
				Name:            "foo",
				InstanceTypeSet: false,
				Index:           -1,
			},
			&ResourceAddress{
				Mode:            config.ManagedResourceMode,
				Type:            "aws_instance",
				Name:            "foo",
				InstanceTypeSet: true,
				InstanceType:    TypePrimary,
				Index:           0,
			},
			true,
		},
		{
			&ResourceAddress{
				Mode:            config.ManagedResourceMode,
				Type:            "aws_instance",
				Name:            "foo",
				InstanceTypeSet: false,
				Index:           -1,
			},
			&ResourceAddress{
				Mode:            config.ManagedResourceMode,
				Type:            "aws_instance",
				Name:            "foo",
				InstanceTypeSet: false,
				Index:           -1,
			},
			true,
		},
		{
			&ResourceAddress{
				InstanceTypeSet: false,
				Index:           -1,
			},
			&ResourceAddress{
				Mode:            config.ManagedResourceMode,
				Type:            "aws_instance",
				Name:            "foo",
				InstanceTypeSet: false,
				Index:           -1,
			},
			true,
		},
		{
			&ResourceAddress{
				InstanceTypeSet: false,
				Index:           -1,
			},
			&ResourceAddress{
				Path:            []string{"bar"},
				Mode:            config.ManagedResourceMode,
				Type:            "aws_instance",
				Name:            "foo",
				InstanceTypeSet: false,
				Index:           -1,
			},
			true,
		},
		{
			&ResourceAddress{
				Path:            []string{"bar"},
				InstanceTypeSet: false,
				Index:           -1,
			},
			&ResourceAddress{
				Path:            []string{"bar"},
				Mode:            config.ManagedResourceMode,
				Type:            "aws_instance",
				Name:            "foo",
				InstanceTypeSet: false,
				Index:           -1,
			},
			true,
		},
		{
			&ResourceAddress{
				Path:            []string{"bar"},
				InstanceTypeSet: false,
				Index:           -1,
			},
			&ResourceAddress{
				Path:            []string{"bar", "baz"},
				Mode:            config.ManagedResourceMode,
				Type:            "aws_instance",
				Name:            "foo",
				InstanceTypeSet: false,
				Index:           -1,
			},
			true,
		},
		{
			&ResourceAddress{
				Path:            []string{"bar"},
				InstanceTypeSet: false,
				Index:           -1,
			},
			&ResourceAddress{
				Path:            []string{"bar", "baz"},
				InstanceTypeSet: false,
				Index:           -1,
			},
			true,
		},
		{
			&ResourceAddress{
				Path:            []string{"bar"},
				InstanceTypeSet: false,
				Index:           -1,
			},
			&ResourceAddress{
				Path:            []string{"bar", "baz", "foo", "pizza"},
				InstanceTypeSet: false,
				Index:           -1,
			},
			true,
		},

		{
			&ResourceAddress{
				Mode:            config.ManagedResourceMode,
				Type:            "aws_instance",
				Name:            "bar",
				InstanceTypeSet: true,
				InstanceType:    TypePrimary,
				Index:           0,
			},
			&ResourceAddress{
				Mode:            config.ManagedResourceMode,
				Type:            "aws_instance",
				Name:            "foo",
				InstanceTypeSet: true,
				InstanceType:    TypePrimary,
				Index:           0,
			},
			false,
		},
		{
			&ResourceAddress{
				Mode:            config.ManagedResourceMode,
				Type:            "aws_instance",
				Name:            "foo",
				InstanceTypeSet: true,
				InstanceType:    TypePrimary,
				Index:           0,
			},
			&ResourceAddress{
				Mode:            config.DataResourceMode,
				Type:            "aws_instance",
				Name:            "foo",
				InstanceTypeSet: true,
				InstanceType:    TypePrimary,
				Index:           0,
			},
			false,
		},
		{
			&ResourceAddress{
				Path:            []string{"bar"},
				InstanceTypeSet: false,
				Index:           -1,
			},
			&ResourceAddress{
				Path:            []string{"baz"},
				Mode:            config.ManagedResourceMode,
				Type:            "aws_instance",
				Name:            "foo",
				InstanceTypeSet: false,
				Index:           -1,
			},
			false,
		},
		{
			&ResourceAddress{
				Path:            []string{"bar"},
				InstanceTypeSet: false,
				Index:           -1,
			},
			&ResourceAddress{
				Path:            []string{"baz", "bar"},
				Mode:            config.ManagedResourceMode,
				Type:            "aws_instance",
				Name:            "foo",
				InstanceTypeSet: false,
				Index:           -1,
			},
			false,
		},
		{
			&ResourceAddress{
				Mode:            config.ManagedResourceMode,
				Type:            "aws_instance",
				Name:            "foo",
				InstanceTypeSet: true,
				InstanceType:    TypePrimary,
				Index:           0,
			},
			&ResourceAddress{
				Mode:            config.ManagedResourceMode,
				Type:            "aws_instance",
				Name:            "foo",
				InstanceTypeSet: false,
				Index:           0,
			},
			false,
		},
		{
			&ResourceAddress{
				Path:            []string{"bar", "baz"},
				InstanceTypeSet: false,
				Index:           -1,
			},
			&ResourceAddress{
				Path:            []string{"bar"},
				InstanceTypeSet: false,
				Index:           -1,
			},
			false,
		},
		{
			&ResourceAddress{
				Type:         "aws_instance",
				Name:         "foo",
				Index:        1,
				InstanceType: TypePrimary,
				Mode:         config.ManagedResourceMode,
			},
			&ResourceAddress{
				Type:         "aws_instance",
				Name:         "foo",
				Index:        -1,
				InstanceType: TypePrimary,
				Mode:         config.ManagedResourceMode,
			},
			false,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%s contains %s", test.Address, test.Other), func(t *testing.T) {
			got := test.Address.Contains(test.Other)
			if got != test.Want {
				t.Errorf(
					"wrong result\nrecv:  %s\ngiven: %s\ngot:   %#v\nwant:  %#v",
					test.Address, test.Other,
					got, test.Want,
				)
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

func TestResourceAddressHasResourceSpec(t *testing.T) {
	cases := []struct {
		Input string
		Want  bool
	}{
		{
			"module.foo",
			false,
		},
		{
			"module.foo.module.bar",
			false,
		},
		{
			"null_resource.baz",
			true,
		},
		{
			"null_resource.baz[0]",
			true,
		},
		{
			"data.null_data_source.baz",
			true,
		},
		{
			"data.null_data_source.baz[0]",
			true,
		},
		{
			"module.foo.null_resource.baz",
			true,
		},
		{
			"module.foo.data.null_data_source.baz",
			true,
		},
		{
			"module.foo.module.bar.null_resource.baz",
			true,
		},
	}

	for _, test := range cases {
		t.Run(test.Input, func(t *testing.T) {
			addr, err := ParseResourceAddress(test.Input)
			if err != nil {
				t.Fatalf("error parsing address: %s", err)
			}
			got := addr.HasResourceSpec()
			if got != test.Want {
				t.Fatalf("%q: wrong result %#v; want %#v", test.Input, got, test.Want)
			}
		})
	}
}

func TestResourceAddressWholeModuleAddress(t *testing.T) {
	cases := []struct {
		Input string
		Want  string
	}{
		{
			"module.foo",
			"module.foo",
		},
		{
			"module.foo.module.bar",
			"module.foo.module.bar",
		},
		{
			"null_resource.baz",
			"",
		},
		{
			"null_resource.baz[0]",
			"",
		},
		{
			"data.null_data_source.baz",
			"",
		},
		{
			"data.null_data_source.baz[0]",
			"",
		},
		{
			"module.foo.null_resource.baz",
			"module.foo",
		},
		{
			"module.foo.data.null_data_source.baz",
			"module.foo",
		},
		{
			"module.foo.module.bar.null_resource.baz",
			"module.foo.module.bar",
		},
	}

	for _, test := range cases {
		t.Run(test.Input, func(t *testing.T) {
			addr, err := ParseResourceAddress(test.Input)
			if err != nil {
				t.Fatalf("error parsing address: %s", err)
			}
			gotAddr := addr.WholeModuleAddress()
			got := gotAddr.String()
			if got != test.Want {
				t.Fatalf("%q: wrong result %#v; want %#v", test.Input, got, test.Want)
			}
		})
	}
}

func TestResourceAddressMatchesResourceConfig(t *testing.T) {
	root := []string(nil)
	child := []string{"child"}
	grandchild := []string{"child", "grandchild"}
	irrelevant := []string{"irrelevant"}

	tests := []struct {
		Addr       *ResourceAddress
		ModulePath []string
		Resource   *configs.Resource
		Want       bool
	}{
		{
			&ResourceAddress{
				Mode:  config.ManagedResourceMode,
				Type:  "null_resource",
				Name:  "baz",
				Index: -1,
			},
			root,
			&configs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "null_resource",
				Name: "baz",
			},
			true,
		},
		{
			&ResourceAddress{
				Path:  []string{"child"},
				Mode:  config.ManagedResourceMode,
				Type:  "null_resource",
				Name:  "baz",
				Index: -1,
			},
			child,
			&configs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "null_resource",
				Name: "baz",
			},
			true,
		},
		{
			&ResourceAddress{
				Path:  []string{"child", "grandchild"},
				Mode:  config.ManagedResourceMode,
				Type:  "null_resource",
				Name:  "baz",
				Index: -1,
			},
			grandchild,
			&configs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "null_resource",
				Name: "baz",
			},
			true,
		},
		{
			&ResourceAddress{
				Path:  []string{"child"},
				Index: -1,
			},
			child,
			&configs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "null_resource",
				Name: "baz",
			},
			true,
		},
		{
			&ResourceAddress{
				Path:  []string{"child", "grandchild"},
				Index: -1,
			},
			grandchild,
			&configs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "null_resource",
				Name: "baz",
			},
			true,
		},
		{
			&ResourceAddress{
				Mode:  config.DataResourceMode,
				Type:  "null_resource",
				Name:  "baz",
				Index: -1,
			},
			irrelevant,
			&configs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "null_resource",
				Name: "baz",
			},
			false,
		},
		{
			&ResourceAddress{
				Mode:  config.ManagedResourceMode,
				Type:  "null_resource",
				Name:  "baz",
				Index: -1,
			},
			irrelevant,
			&configs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "null_resource",
				Name: "pizza",
			},
			false,
		},
		{
			&ResourceAddress{
				Mode:  config.ManagedResourceMode,
				Type:  "null_resource",
				Name:  "baz",
				Index: -1,
			},
			irrelevant,
			&configs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "aws_instance",
				Name: "baz",
			},
			false,
		},
		{
			&ResourceAddress{
				Path:  []string{"child", "grandchild"},
				Mode:  config.ManagedResourceMode,
				Type:  "null_resource",
				Name:  "baz",
				Index: -1,
			},
			child,
			&configs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "null_resource",
				Name: "baz",
			},
			false,
		},
		{
			&ResourceAddress{
				Path:  []string{"child"},
				Mode:  config.ManagedResourceMode,
				Type:  "null_resource",
				Name:  "baz",
				Index: -1,
			},
			grandchild,
			&configs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "null_resource",
				Name: "baz",
			},
			false,
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("%02d-%s", i, test.Addr), func(t *testing.T) {
			got := test.Addr.MatchesResourceConfig(test.ModulePath, test.Resource)
			if got != test.Want {
				t.Errorf(
					"wrong result\naddr: %s\nmod:  %#v\nrsrc: %#v\ngot:  %#v\nwant: %#v",
					test.Addr, test.ModulePath, test.Resource, got, test.Want,
				)
			}
		})
	}
}

func TestResourceAddressLess(t *testing.T) {
	tests := []struct {
		A    string
		B    string
		Want bool
	}{
		{
			"foo.bar",
			"module.baz.foo.bar",
			true,
		},
		{
			"module.baz.foo.bar",
			"zzz.bar", // would sort after "module" in lexicographical sort
			false,
		},
		{
			"module.baz.foo.bar",
			"module.baz.foo.bar",
			false,
		},
		{
			"module.baz.foo.bar",
			"module.boz.foo.bar",
			true,
		},
		{
			"module.boz.foo.bar",
			"module.baz.foo.bar",
			false,
		},
		{
			"a.b",
			"b.c",
			true,
		},
		{
			"a.b",
			"a.c",
			true,
		},
		{
			"c.b",
			"b.c",
			false,
		},
		{
			"a.b[9]",
			"a.b[10]",
			true,
		},
		{
			"b.b[9]",
			"a.b[10]",
			false,
		},
		{
			"a.b",
			"a.b.deposed",
			true,
		},
		{
			"a.b.tainted",
			"a.b.deposed",
			true,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%s < %s", test.A, test.B), func(t *testing.T) {
			addrA, err := ParseResourceAddress(test.A)
			if err != nil {
				t.Fatal(err)
			}
			addrB, err := ParseResourceAddress(test.B)
			if err != nil {
				t.Fatal(err)
			}
			got := addrA.Less(addrB)
			invGot := addrB.Less(addrA)
			if got != test.Want {
				t.Errorf(
					"wrong result\ntest: %s < %s\ngot:  %#v\nwant: %#v",
					test.A, test.B, got, test.Want,
				)
			}
			if test.A != test.B { // inverse test doesn't apply when equal
				if invGot != !test.Want {
					t.Errorf(
						"wrong inverse result\ntest: %s < %s\ngot:  %#v\nwant: %#v",
						test.B, test.A, invGot, !test.Want,
					)
				}
			} else {
				if invGot != test.Want {
					t.Errorf(
						"wrong inverse result\ntest: %s < %s\ngot:  %#v\nwant: %#v",
						test.B, test.A, invGot, test.Want,
					)
				}
			}
		})
	}
}
