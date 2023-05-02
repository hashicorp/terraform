// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package addrs

import (
	"testing"

	"github.com/go-test/deep"
	svchost "github.com/hashicorp/terraform-svchost"
)

func TestProviderString(t *testing.T) {
	tests := []struct {
		Input Provider
		Want  string
	}{
		{
			Provider{
				Type:      "test",
				Hostname:  DefaultProviderRegistryHost,
				Namespace: "hashicorp",
			},
			NewDefaultProvider("test").String(),
		},
		{
			Provider{
				Type:      "test-beta",
				Hostname:  DefaultProviderRegistryHost,
				Namespace: "hashicorp",
			},
			NewDefaultProvider("test-beta").String(),
		},
		{
			Provider{
				Type:      "test",
				Hostname:  "registry.terraform.com",
				Namespace: "hashicorp",
			},
			"registry.terraform.com/hashicorp/test",
		},
		{
			Provider{
				Type:      "test",
				Hostname:  DefaultProviderRegistryHost,
				Namespace: "othercorp",
			},
			DefaultProviderRegistryHost.ForDisplay() + "/othercorp/test",
		},
	}

	for _, test := range tests {
		got := test.Input.String()
		if got != test.Want {
			t.Errorf("wrong result for %s\n", test.Input.String())
		}
	}
}

func TestProviderLegacyString(t *testing.T) {
	tests := []struct {
		Input Provider
		Want  string
	}{
		{
			Provider{
				Type:      "test",
				Hostname:  DefaultProviderRegistryHost,
				Namespace: LegacyProviderNamespace,
			},
			"test",
		},
		{
			Provider{
				Type:      "terraform",
				Hostname:  BuiltInProviderHost,
				Namespace: BuiltInProviderNamespace,
			},
			"terraform",
		},
	}

	for _, test := range tests {
		got := test.Input.LegacyString()
		if got != test.Want {
			t.Errorf("wrong result for %s\ngot:  %s\nwant: %s", test.Input.String(), got, test.Want)
		}
	}
}

func TestProviderDisplay(t *testing.T) {
	tests := []struct {
		Input Provider
		Want  string
	}{
		{
			Provider{
				Type:      "test",
				Hostname:  DefaultProviderRegistryHost,
				Namespace: "hashicorp",
			},
			"hashicorp/test",
		},
		{
			Provider{
				Type:      "test",
				Hostname:  "registry.terraform.com",
				Namespace: "hashicorp",
			},
			"registry.terraform.com/hashicorp/test",
		},
		{
			Provider{
				Type:      "test",
				Hostname:  DefaultProviderRegistryHost,
				Namespace: "othercorp",
			},
			"othercorp/test",
		},
	}

	for _, test := range tests {
		got := test.Input.ForDisplay()
		if got != test.Want {
			t.Errorf("wrong result for %s\n", test.Input.String())
		}
	}
}

func TestProviderIsDefaultProvider(t *testing.T) {
	tests := []struct {
		Input Provider
		Want  bool
	}{
		{
			Provider{
				Type:      "test",
				Hostname:  DefaultProviderRegistryHost,
				Namespace: "hashicorp",
			},
			true,
		},
		{
			Provider{
				Type:      "test",
				Hostname:  "registry.terraform.com",
				Namespace: "hashicorp",
			},
			false,
		},
		{
			Provider{
				Type:      "test",
				Hostname:  DefaultProviderRegistryHost,
				Namespace: "othercorp",
			},
			false,
		},
	}

	for _, test := range tests {
		got := IsDefaultProvider(test.Input)
		if got != test.Want {
			t.Errorf("wrong result for %s\n", test.Input.String())
		}
	}
}

func TestProviderIsBuiltIn(t *testing.T) {
	tests := []struct {
		Input Provider
		Want  bool
	}{
		{
			Provider{
				Type:      "test",
				Hostname:  BuiltInProviderHost,
				Namespace: BuiltInProviderNamespace,
			},
			true,
		},
		{
			Provider{
				Type:      "terraform",
				Hostname:  BuiltInProviderHost,
				Namespace: BuiltInProviderNamespace,
			},
			true,
		},
		{
			Provider{
				Type:      "test",
				Hostname:  BuiltInProviderHost,
				Namespace: "boop",
			},
			false,
		},
		{
			Provider{
				Type:      "test",
				Hostname:  DefaultProviderRegistryHost,
				Namespace: BuiltInProviderNamespace,
			},
			false,
		},
		{
			Provider{
				Type:      "test",
				Hostname:  DefaultProviderRegistryHost,
				Namespace: "hashicorp",
			},
			false,
		},
		{
			Provider{
				Type:      "test",
				Hostname:  "registry.terraform.com",
				Namespace: "hashicorp",
			},
			false,
		},
		{
			Provider{
				Type:      "test",
				Hostname:  DefaultProviderRegistryHost,
				Namespace: "othercorp",
			},
			false,
		},
	}

	for _, test := range tests {
		got := test.Input.IsBuiltIn()
		if got != test.Want {
			t.Errorf("wrong result for %s\ngot:  %#v\nwant: %#v", test.Input.String(), got, test.Want)
		}
	}
}

func TestProviderIsLegacy(t *testing.T) {
	tests := []struct {
		Input Provider
		Want  bool
	}{
		{
			Provider{
				Type:      "test",
				Hostname:  DefaultProviderRegistryHost,
				Namespace: LegacyProviderNamespace,
			},
			true,
		},
		{
			Provider{
				Type:      "test",
				Hostname:  "registry.terraform.com",
				Namespace: LegacyProviderNamespace,
			},
			false,
		},
		{
			Provider{
				Type:      "test",
				Hostname:  DefaultProviderRegistryHost,
				Namespace: "hashicorp",
			},
			false,
		},
	}

	for _, test := range tests {
		got := test.Input.IsLegacy()
		if got != test.Want {
			t.Errorf("wrong result for %s\n", test.Input.String())
		}
	}
}

func TestParseProviderSourceStr(t *testing.T) {
	tests := map[string]struct {
		Want Provider
		Err  bool
	}{
		"registry.terraform.io/hashicorp/aws": {
			Provider{
				Type:      "aws",
				Namespace: "hashicorp",
				Hostname:  DefaultProviderRegistryHost,
			},
			false,
		},
		"registry.Terraform.io/HashiCorp/AWS": {
			Provider{
				Type:      "aws",
				Namespace: "hashicorp",
				Hostname:  DefaultProviderRegistryHost,
			},
			false,
		},
		"hashicorp/aws": {
			Provider{
				Type:      "aws",
				Namespace: "hashicorp",
				Hostname:  DefaultProviderRegistryHost,
			},
			false,
		},
		"HashiCorp/AWS": {
			Provider{
				Type:      "aws",
				Namespace: "hashicorp",
				Hostname:  DefaultProviderRegistryHost,
			},
			false,
		},
		"aws": {
			Provider{
				Type:      "aws",
				Namespace: "hashicorp",
				Hostname:  DefaultProviderRegistryHost,
			},
			false,
		},
		"AWS": {
			Provider{
				Type:      "aws",
				Namespace: "hashicorp",
				Hostname:  DefaultProviderRegistryHost,
			},
			false,
		},
		"example.com/foo-bar/baz-boop": {
			Provider{
				Type:      "baz-boop",
				Namespace: "foo-bar",
				Hostname:  svchost.Hostname("example.com"),
			},
			false,
		},
		"foo-bar/baz-boop": {
			Provider{
				Type:      "baz-boop",
				Namespace: "foo-bar",
				Hostname:  DefaultProviderRegistryHost,
			},
			false,
		},
		"localhost:8080/foo/bar": {
			Provider{
				Type:      "bar",
				Namespace: "foo",
				Hostname:  svchost.Hostname("localhost:8080"),
			},
			false,
		},
		"example.com/too/many/parts/here": {
			Provider{},
			true,
		},
		"/too///many//slashes": {
			Provider{},
			true,
		},
		"///": {
			Provider{},
			true,
		},
		"/ / /": { // empty strings
			Provider{},
			true,
		},
		"badhost!/hashicorp/aws": {
			Provider{},
			true,
		},
		"example.com/badnamespace!/aws": {
			Provider{},
			true,
		},
		"example.com/bad--namespace/aws": {
			Provider{},
			true,
		},
		"example.com/-badnamespace/aws": {
			Provider{},
			true,
		},
		"example.com/badnamespace-/aws": {
			Provider{},
			true,
		},
		"example.com/bad.namespace/aws": {
			Provider{},
			true,
		},
		"example.com/hashicorp/badtype!": {
			Provider{},
			true,
		},
		"example.com/hashicorp/bad--type": {
			Provider{},
			true,
		},
		"example.com/hashicorp/-badtype": {
			Provider{},
			true,
		},
		"example.com/hashicorp/badtype-": {
			Provider{},
			true,
		},
		"example.com/hashicorp/bad.type": {
			Provider{},
			true,
		},

		// We forbid the terraform- prefix both because it's redundant to
		// include "terraform" in a Terraform provider name and because we use
		// the longer prefix terraform-provider- to hint for users who might be
		// accidentally using the git repository name or executable file name
		// instead of the provider type.
		"example.com/hashicorp/terraform-provider-bad": {
			Provider{},
			true,
		},
		"example.com/hashicorp/terraform-bad": {
			Provider{},
			true,
		},
	}

	for name, test := range tests {
		got, diags := ParseProviderSourceString(name)
		for _, problem := range deep.Equal(got, test.Want) {
			t.Errorf(problem)
		}
		if len(diags) > 0 {
			if test.Err == false {
				t.Errorf("got error, expected success")
			}
		} else {
			if test.Err {
				t.Errorf("got success, expected error")
			}
		}
	}
}

func TestParseProviderPart(t *testing.T) {
	tests := map[string]struct {
		Want  string
		Error string
	}{
		`foo`: {
			`foo`,
			``,
		},
		`FOO`: {
			`foo`,
			``,
		},
		`Foo`: {
			`foo`,
			``,
		},
		`abc-123`: {
			`abc-123`,
			``,
		},
		`Испытание`: {
			`испытание`,
			``,
		},
		`münchen`: { // this is a precomposed u with diaeresis
			`münchen`, // this is a precomposed u with diaeresis
			``,
		},
		`münchen`: { // this is a separate u and combining diaeresis
			`münchen`, // this is a precomposed u with diaeresis
			``,
		},
		`abc--123`: {
			``,
			`cannot use multiple consecutive dashes`,
		},
		`xn--80akhbyknj4f`: { // this is the punycode form of "испытание", but we don't accept punycode here
			``,
			`cannot use multiple consecutive dashes`,
		},
		`abc.123`: {
			``,
			`dots are not allowed`,
		},
		`-abc123`: {
			``,
			`must contain only letters, digits, and dashes, and may not use leading or trailing dashes`,
		},
		`abc123-`: {
			``,
			`must contain only letters, digits, and dashes, and may not use leading or trailing dashes`,
		},
		``: {
			``,
			`must have at least one character`,
		},
	}

	for given, test := range tests {
		t.Run(given, func(t *testing.T) {
			got, err := ParseProviderPart(given)
			if test.Error != "" {
				if err == nil {
					t.Errorf("unexpected success\ngot:  %s\nwant: %s", err, test.Error)
				} else if got := err.Error(); got != test.Error {
					t.Errorf("wrong error\ngot:  %s\nwant: %s", got, test.Error)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error\ngot:  %s\nwant: <nil>", err)
				} else if got != test.Want {
					t.Errorf("wrong result\ngot:  %s\nwant: %s", got, test.Want)
				}
			}
		})
	}
}

func TestProviderEquals(t *testing.T) {
	tests := []struct {
		InputP Provider
		OtherP Provider
		Want   bool
	}{
		{
			NewProvider(DefaultProviderRegistryHost, "foo", "test"),
			NewProvider(DefaultProviderRegistryHost, "foo", "test"),
			true,
		},
		{
			NewProvider(DefaultProviderRegistryHost, "foo", "test"),
			NewProvider(DefaultProviderRegistryHost, "bar", "test"),
			false,
		},
		{
			NewProvider(DefaultProviderRegistryHost, "foo", "test"),
			NewProvider(DefaultProviderRegistryHost, "foo", "my-test"),
			false,
		},
		{
			NewProvider(DefaultProviderRegistryHost, "foo", "test"),
			NewProvider("example.com", "foo", "test"),
			false,
		},
	}
	for _, test := range tests {
		t.Run(test.InputP.String(), func(t *testing.T) {
			got := test.InputP.Equals(test.OtherP)
			if got != test.Want {
				t.Errorf("wrong result\ngot:  %v\nwant: %v", got, test.Want)
			}
		})
	}
}
