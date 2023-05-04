// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package funcs

import (
	"fmt"
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestCidrHost(t *testing.T) {
	tests := []struct {
		Prefix  cty.Value
		Hostnum cty.Value
		Want    cty.Value
		Err     bool
	}{
		{
			cty.StringVal("192.168.1.0/24"),
			cty.NumberIntVal(5),
			cty.StringVal("192.168.1.5"),
			false,
		},
		{
			cty.StringVal("192.168.1.0/24"),
			cty.NumberIntVal(-5),
			cty.StringVal("192.168.1.251"),
			false,
		},
		{
			cty.StringVal("192.168.1.0/24"),
			cty.NumberIntVal(-256),
			cty.StringVal("192.168.1.0"),
			false,
		},
		{
			// We inadvertently inherited a pre-Go1.17 standard library quirk
			// if parsing zero-prefix parts as decimal rather than octal.
			// Go 1.17 resolved that quirk by making zero-prefix invalid, but
			// we've preserved our existing behavior for backward compatibility,
			// on the grounds that these functions are for generating addresses
			// rather than validating or processing them. We do always generate
			// a canonical result regardless of the input, though.
			cty.StringVal("010.001.0.0/24"),
			cty.NumberIntVal(6),
			cty.StringVal("10.1.0.6"),
			false,
		},
		{
			cty.StringVal("192.168.1.0/30"),
			cty.NumberIntVal(255),
			cty.UnknownVal(cty.String),
			true, // 255 doesn't fit in two bits
		},
		{
			cty.StringVal("192.168.1.0/30"),
			cty.NumberIntVal(-255),
			cty.UnknownVal(cty.String),
			true, // 255 doesn't fit in two bits
		},
		{
			cty.StringVal("not-a-cidr"),
			cty.NumberIntVal(6),
			cty.UnknownVal(cty.String),
			true, // not a valid CIDR mask
		},
		{
			cty.StringVal("10.256.0.0/8"),
			cty.NumberIntVal(6),
			cty.UnknownVal(cty.String),
			true, // can't have an octet >255
		},
		{ // fractions are Not Ok
			cty.StringVal("10.256.0.0/8"),
			cty.NumberFloatVal(.75),
			cty.UnknownVal(cty.String),
			true,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("cidrhost(%#v, %#v)", test.Prefix, test.Hostnum), func(t *testing.T) {
			got, err := CidrHost(test.Prefix, test.Hostnum)

			if test.Err {
				if err == nil {
					t.Fatal("succeeded; want error")
				}
				return
			} else if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if !got.RawEquals(test.Want) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.Want)
			}
		})
	}
}

func TestCidrNetmask(t *testing.T) {
	tests := []struct {
		Prefix cty.Value
		Want   cty.Value
		Err    bool
	}{
		{
			cty.StringVal("192.168.1.0/24"),
			cty.StringVal("255.255.255.0"),
			false,
		},
		{
			cty.StringVal("192.168.1.0/32"),
			cty.StringVal("255.255.255.255"),
			false,
		},
		{
			cty.StringVal("0.0.0.0/0"),
			cty.StringVal("0.0.0.0"),
			false,
		},
		{
			// We inadvertently inherited a pre-Go1.17 standard library quirk
			// if parsing zero-prefix parts as decimal rather than octal.
			// Go 1.17 resolved that quirk by making zero-prefix invalid, but
			// we've preserved our existing behavior for backward compatibility,
			// on the grounds that these functions are for generating addresses
			// rather than validating or processing them.
			cty.StringVal("010.001.0.0/24"),
			cty.StringVal("255.255.255.0"),
			false,
		},
		{
			cty.StringVal("not-a-cidr"),
			cty.UnknownVal(cty.String),
			true, // not a valid CIDR mask
		},
		{
			cty.StringVal("110.256.0.0/8"),
			cty.UnknownVal(cty.String),
			true, // can't have an octet >255
		},
		{
			cty.StringVal("1::/64"),
			cty.UnknownVal(cty.String),
			true, // IPv6 is invalid
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("cidrnetmask(%#v)", test.Prefix), func(t *testing.T) {
			got, err := CidrNetmask(test.Prefix)

			if test.Err {
				if err == nil {
					t.Fatal("succeeded; want error")
				}
				return
			} else if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if !got.RawEquals(test.Want) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.Want)
			}
		})
	}
}

func TestCidrSubnet(t *testing.T) {
	tests := []struct {
		Prefix  cty.Value
		Newbits cty.Value
		Netnum  cty.Value
		Want    cty.Value
		Err     bool
	}{
		{
			cty.StringVal("192.168.2.0/20"),
			cty.NumberIntVal(4),
			cty.NumberIntVal(6),
			cty.StringVal("192.168.6.0/24"),
			false,
		},
		{
			cty.StringVal("fe80::/48"),
			cty.NumberIntVal(16),
			cty.NumberIntVal(6),
			cty.StringVal("fe80:0:0:6::/64"),
			false,
		},
		{ // IPv4 address encoded in IPv6 syntax gets normalized
			cty.StringVal("::ffff:192.168.0.0/112"),
			cty.NumberIntVal(8),
			cty.NumberIntVal(6),
			cty.StringVal("192.168.6.0/24"),
			false,
		},
		{
			cty.StringVal("fe80::/48"),
			cty.NumberIntVal(33),
			cty.NumberIntVal(6),
			cty.StringVal("fe80::3:0:0:0/81"),
			false,
		},
		{
			// We inadvertently inherited a pre-Go1.17 standard library quirk
			// if parsing zero-prefix parts as decimal rather than octal.
			// Go 1.17 resolved that quirk by making zero-prefix invalid, but
			// we've preserved our existing behavior for backward compatibility,
			// on the grounds that these functions are for generating addresses
			// rather than validating or processing them. We do always generate
			// a canonical result regardless of the input, though.
			cty.StringVal("010.001.0.0/24"),
			cty.NumberIntVal(4),
			cty.NumberIntVal(1),
			cty.StringVal("10.1.0.16/28"),
			false,
		},
		{ // not enough bits left
			cty.StringVal("192.168.0.0/30"),
			cty.NumberIntVal(4),
			cty.NumberIntVal(6),
			cty.UnknownVal(cty.String),
			true,
		},
		{ // can't encode 16 in 2 bits
			cty.StringVal("192.168.0.0/168"),
			cty.NumberIntVal(2),
			cty.NumberIntVal(16),
			cty.UnknownVal(cty.String),
			true,
		},
		{ // not a valid CIDR mask
			cty.StringVal("not-a-cidr"),
			cty.NumberIntVal(4),
			cty.NumberIntVal(6),
			cty.UnknownVal(cty.String),
			true,
		},
		{ // can't have an octet >255
			cty.StringVal("10.256.0.0/8"),
			cty.NumberIntVal(4),
			cty.NumberIntVal(6),
			cty.UnknownVal(cty.String),
			true,
		},
		{ // fractions are Not Ok
			cty.StringVal("10.256.0.0/8"),
			cty.NumberFloatVal(2.0 / 3.0),
			cty.NumberFloatVal(.75),
			cty.UnknownVal(cty.String),
			true,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("cidrsubnet(%#v, %#v, %#v)", test.Prefix, test.Newbits, test.Netnum), func(t *testing.T) {
			got, err := CidrSubnet(test.Prefix, test.Newbits, test.Netnum)

			if test.Err {
				if err == nil {
					t.Fatal("succeeded; want error")
				}
				return
			} else if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if !got.RawEquals(test.Want) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.Want)
			}
		})
	}
}
func TestCidrSubnets(t *testing.T) {
	tests := []struct {
		Prefix  cty.Value
		Newbits []cty.Value
		Want    cty.Value
		Err     string
	}{
		{
			cty.StringVal("10.0.0.0/21"),
			[]cty.Value{
				cty.NumberIntVal(3),
				cty.NumberIntVal(3),
				cty.NumberIntVal(3),
				cty.NumberIntVal(4),
				cty.NumberIntVal(4),
				cty.NumberIntVal(4),
				cty.NumberIntVal(7),
				cty.NumberIntVal(7),
				cty.NumberIntVal(7),
			},
			cty.ListVal([]cty.Value{
				cty.StringVal("10.0.0.0/24"),
				cty.StringVal("10.0.1.0/24"),
				cty.StringVal("10.0.2.0/24"),
				cty.StringVal("10.0.3.0/25"),
				cty.StringVal("10.0.3.128/25"),
				cty.StringVal("10.0.4.0/25"),
				cty.StringVal("10.0.4.128/28"),
				cty.StringVal("10.0.4.144/28"),
				cty.StringVal("10.0.4.160/28"),
			}),
			``,
		},
		{
			// We inadvertently inherited a pre-Go1.17 standard library quirk
			// if parsing zero-prefix parts as decimal rather than octal.
			// Go 1.17 resolved that quirk by making zero-prefix invalid, but
			// we've preserved our existing behavior for backward compatibility,
			// on the grounds that these functions are for generating addresses
			// rather than validating or processing them. We do always generate
			// a canonical result regardless of the input, though.
			cty.StringVal("010.0.0.0/21"),
			[]cty.Value{
				cty.NumberIntVal(3),
			},
			cty.ListVal([]cty.Value{
				cty.StringVal("10.0.0.0/24"),
			}),
			``,
		},
		{
			cty.StringVal("10.0.0.0/30"),
			[]cty.Value{
				cty.NumberIntVal(1),
				cty.NumberIntVal(3),
			},
			cty.UnknownVal(cty.List(cty.String)),
			`would extend prefix to 33 bits, which is too long for an IPv4 address`,
		},
		{
			cty.StringVal("10.0.0.0/8"),
			[]cty.Value{
				cty.NumberIntVal(1),
				cty.NumberIntVal(1),
				cty.NumberIntVal(1),
			},
			cty.UnknownVal(cty.List(cty.String)),
			`not enough remaining address space for a subnet with a prefix of 9 bits after 10.128.0.0/9`,
		},
		{
			cty.StringVal("10.0.0.0/8"),
			[]cty.Value{
				cty.NumberIntVal(1),
				cty.NumberIntVal(0),
			},
			cty.UnknownVal(cty.List(cty.String)),
			`must extend prefix by at least one bit`,
		},
		{
			cty.StringVal("10.0.0.0/8"),
			[]cty.Value{
				cty.NumberIntVal(1),
				cty.NumberIntVal(-1),
			},
			cty.UnknownVal(cty.List(cty.String)),
			`must extend prefix by at least one bit`,
		},
		{
			cty.StringVal("fe80::/48"),
			[]cty.Value{
				cty.NumberIntVal(1),
				cty.NumberIntVal(33),
			},
			cty.UnknownVal(cty.List(cty.String)),
			`may not extend prefix by more than 32 bits`,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("cidrsubnets(%#v, %#v)", test.Prefix, test.Newbits), func(t *testing.T) {
			got, err := CidrSubnets(test.Prefix, test.Newbits...)
			wantErr := test.Err != ""

			if wantErr {
				if err == nil {
					t.Fatal("succeeded; want error")
				}
				if err.Error() != test.Err {
					t.Fatalf("wrong error\ngot:  %s\nwant: %s", err.Error(), test.Err)
				}
				return
			} else if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if !got.RawEquals(test.Want) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.Want)
			}
		})
	}
}
