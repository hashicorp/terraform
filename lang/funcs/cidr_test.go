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
			cty.StringVal("1::/64"),
			cty.StringVal("ffff:ffff:ffff:ffff::"),
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
			cty.StringVal("fe80:0:0:6::/64"),
			true,
		},
		{ // not a valid CIDR mask
			cty.StringVal("not-a-cidr"),
			cty.NumberIntVal(4),
			cty.NumberIntVal(6),
			cty.StringVal("fe80:0:0:6::/64"),
			true,
		},
		{ // can't have an octet >255
			cty.StringVal("10.256.0.0/8"),
			cty.NumberIntVal(4),
			cty.NumberIntVal(6),
			cty.StringVal("fe80:0:0:6::/64"),
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
