// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package funcs

import (
	"fmt"
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/lang/marks"
)

func TestBase64Decode(t *testing.T) {
	tests := []struct {
		String cty.Value
		Want   cty.Value
		Err    bool
	}{
		{
			cty.StringVal("YWJjMTIzIT8kKiYoKSctPUB+"),
			cty.StringVal("abc123!?$*&()'-=@~"),
			false,
		},
		{
			cty.StringVal("YWJjMTIzIT8kKiYoKSctPUB+").Mark(marks.Sensitive),
			cty.StringVal("abc123!?$*&()'-=@~").Mark(marks.Sensitive),
			false,
		},
		{ // Invalid base64 data decoding
			cty.StringVal("this-is-an-invalid-base64-data"),
			cty.UnknownVal(cty.String),
			true,
		},
		{ // Invalid utf-8
			cty.StringVal("\xc3\x28"),
			cty.UnknownVal(cty.String),
			true,
		},
		// unknown marked
		{
			cty.UnknownVal(cty.String).Mark("a").Mark("b"),
			cty.UnknownVal(cty.String).RefineNotNull().Mark("a").Mark("b"),
			false,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("base64decode(%#v)", test.String), func(t *testing.T) {
			got, err := Base64DecodeFunc.Call([]cty.Value{test.String})

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

func TestBase64Decode_error(t *testing.T) {
	tests := map[string]struct {
		String  cty.Value
		WantErr string
	}{
		"invalid base64": {
			cty.StringVal("dfg"),
			`failed to decode base64 data "dfg"`,
		},
		"sensitive invalid base64": {
			cty.StringVal("dfg").Mark(marks.Sensitive),
			`failed to decode base64 data (sensitive value)`,
		},
		"invalid utf-8": {
			cty.StringVal("whee"),
			"the result of decoding the provided string is not valid UTF-8",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := Base64DecodeFunc.Call([]cty.Value{test.String})

			if err == nil {
				t.Fatal("succeeded; want error")
			}

			if err.Error() != test.WantErr {
				t.Errorf("wrong error result\ngot:  %#v\nwant: %#v", err.Error(), test.WantErr)
			}
		})
	}
}

func TestBase64Encode(t *testing.T) {
	tests := []struct {
		String cty.Value
		Want   cty.Value
		Err    bool
	}{
		{
			cty.StringVal("abc123!?$*&()'-=@~"),
			cty.StringVal("YWJjMTIzIT8kKiYoKSctPUB+"),
			false,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("base64encode(%#v)", test.String), func(t *testing.T) {
			got, err := Base64EncodeFunc.Call([]cty.Value{test.String})

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

func TestBase64Gzip(t *testing.T) {
	tests := []struct {
		String cty.Value
		Want   cty.Value
		Err    bool
	}{
		{
			cty.StringVal("test"),
			cty.StringVal("H4sIAAAAAAAA/ypJLS4BAAAA//8BAAD//wx+f9gEAAAA"),
			false,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("base64gzip(%#v)", test.String), func(t *testing.T) {
			got, err := Base64GzipFunc.Call([]cty.Value{test.String})

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

func TestURLEncode(t *testing.T) {
	tests := []struct {
		String cty.Value
		Want   cty.Value
		Err    bool
	}{
		{
			cty.StringVal("abc123-_"),
			cty.StringVal("abc123-_"),
			false,
		},
		{
			cty.StringVal("foo:bar@localhost?foo=bar&bar=baz"),
			cty.StringVal("foo%3Abar%40localhost%3Ffoo%3Dbar%26bar%3Dbaz"),
			false,
		},
		{
			cty.StringVal("mailto:email?subject=this+is+my+subject"),
			cty.StringVal("mailto%3Aemail%3Fsubject%3Dthis%2Bis%2Bmy%2Bsubject"),
			false,
		},
		{
			cty.StringVal("foo/bar"),
			cty.StringVal("foo%2Fbar"),
			false,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("urlencode(%#v)", test.String), func(t *testing.T) {
			got, err := URLEncodeFunc.Call([]cty.Value{test.String})

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

func TestBase64TextEncode(t *testing.T) {
	tests := []struct {
		String   cty.Value
		Encoding cty.Value
		Want     cty.Value
		Err      string
	}{
		{
			cty.StringVal("abc123!?$*&()'-=@~"),
			cty.StringVal("UTF-8"),
			cty.StringVal("YWJjMTIzIT8kKiYoKSctPUB+"),
			``,
		},
		{
			cty.StringVal("abc123!?$*&()'-=@~"),
			cty.StringVal("UTF-16LE"),
			cty.StringVal("YQBiAGMAMQAyADMAIQA/ACQAKgAmACgAKQAnAC0APQBAAH4A"),
			``,
		},
		{
			cty.StringVal("abc123!?$*&()'-=@~"),
			cty.StringVal("CP936"),
			cty.StringVal("YWJjMTIzIT8kKiYoKSctPUB+"),
			``,
		},
		{
			cty.StringVal("abc123!?$*&()'-=@~"),
			cty.StringVal("NOT-EXISTS"),
			cty.UnknownVal(cty.String).RefineNotNull(),
			`"NOT-EXISTS" is not a supported IANA encoding name or alias in this Terraform version`,
		},
		{
			cty.StringVal("🤔"),
			cty.StringVal("cp437"),
			cty.UnknownVal(cty.String).RefineNotNull(),
			`the given string contains characters that cannot be represented in IBM437`,
		},
		{
			cty.UnknownVal(cty.String),
			cty.StringVal("windows-1250"),
			cty.UnknownVal(cty.String).RefineNotNull(),
			``,
		},
		{
			cty.StringVal("hello world"),
			cty.UnknownVal(cty.String),
			cty.UnknownVal(cty.String).RefineNotNull(),
			``,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("textencodebase64(%#v, %#v)", test.String, test.Encoding), func(t *testing.T) {
			got, err := TextEncodeBase64Func.Call([]cty.Value{test.String, test.Encoding})

			if test.Err != "" {
				if err == nil {
					t.Fatal("succeeded; want error")
				}
				if got, want := err.Error(), test.Err; got != want {
					t.Fatalf("wrong error\ngot:  %s\nwant: %s", got, want)
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

func TestBase64TextDecode(t *testing.T) {
	tests := []struct {
		String   cty.Value
		Encoding cty.Value
		Want     cty.Value
		Err      string
	}{
		{
			cty.StringVal("YWJjMTIzIT8kKiYoKSctPUB+"),
			cty.StringVal("UTF-8"),
			cty.StringVal("abc123!?$*&()'-=@~"),
			``,
		},
		{
			cty.StringVal("YQBiAGMAMQAyADMAIQA/ACQAKgAmACgAKQAnAC0APQBAAH4A"),
			cty.StringVal("UTF-16LE"),
			cty.StringVal("abc123!?$*&()'-=@~"),
			``,
		},
		{
			cty.StringVal("YWJjMTIzIT8kKiYoKSctPUB+"),
			cty.StringVal("CP936"),
			cty.StringVal("abc123!?$*&()'-=@~"),
			``,
		},
		{
			cty.StringVal("doesn't matter"),
			cty.StringVal("NOT-EXISTS"),
			cty.UnknownVal(cty.String).RefineNotNull(),
			`"NOT-EXISTS" is not a supported IANA encoding name or alias in this Terraform version`,
		},
		{
			cty.StringVal("<invalid base64>"),
			cty.StringVal("cp437"),
			cty.UnknownVal(cty.String).RefineNotNull(),
			`the given value has an invalid base64 symbol at offset 0`,
		},
		{
			cty.StringVal("gQ=="), // this is 0x81, which is not defined in windows-1250
			cty.StringVal("windows-1250"),
			cty.StringVal("�"),
			`the given string contains symbols that are not defined for windows-1250`,
		},
		{
			// U+FFFD is a character like any other once it is in the source:
			// the decoder has nothing to substitute here, so the value has to
			// come back as it went in.
			cty.StringVal("Y2Fm77+9"), // "caf\ufffd" in UTF-8
			cty.StringVal("UTF-8"),
			cty.StringVal("caf\ufffd"),
			``,
		},
		{
			cty.StringVal("/f8="), // U+FFFD alone in UTF-16LE
			cty.StringVal("UTF-16LE"),
			cty.StringVal("\ufffd"),
			``,
		},
		{
			cty.UnknownVal(cty.String),
			cty.StringVal("windows-1250"),
			cty.UnknownVal(cty.String).RefineNotNull(),
			``,
		},
		{
			cty.StringVal("YQBiAGMAMQAyADMAIQA/ACQAKgAmACgAKQAnAC0APQBAAH4A"),
			cty.UnknownVal(cty.String),
			cty.UnknownVal(cty.String).RefineNotNull(),
			``,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("textdecodebase64(%#v, %#v)", test.String, test.Encoding), func(t *testing.T) {
			got, err := TextDecodeBase64Func.Call([]cty.Value{test.String, test.Encoding})

			if test.Err != "" {
				if err == nil {
					t.Fatal("succeeded; want error")
				}
				if got, want := err.Error(), test.Err; got != want {
					t.Fatalf("wrong error\ngot:  %s\nwant: %s", got, want)
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

// TestBase64TextEncodeDecodeRoundTrip covers the case that made the
// replacement-character check unsound on its own: a string that contains
// U+FFFD itself encodes without complaint, so it has to decode again.
func TestBase64TextEncodeDecodeRoundTrip(t *testing.T) {
	strings := []string{
		"�",
		"caf�",
		"� at the start",
		"plain ascii",
	}
	encodings := []string{"UTF-8", "UTF-16LE"}

	for _, encName := range encodings {
		for _, str := range strings {
			t.Run(fmt.Sprintf("%s/%q", encName, str), func(t *testing.T) {
				enc, err := TextEncodeBase64Func.Call([]cty.Value{cty.StringVal(str), cty.StringVal(encName)})
				if err != nil {
					t.Fatalf("textencodebase64 failed: %s", err)
				}

				got, err := TextDecodeBase64Func.Call([]cty.Value{enc, cty.StringVal(encName)})
				if err != nil {
					t.Fatalf("textdecodebase64 failed: %s", err)
				}

				if want := cty.StringVal(str); !got.RawEquals(want) {
					t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, want)
				}
			})
		}
	}
}
