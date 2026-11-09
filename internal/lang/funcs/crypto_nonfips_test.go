//go:build !fips

// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package funcs

import (
	"fmt"
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestMd5(t *testing.T) {
	tests := []struct {
		String cty.Value
		Want   cty.Value
		Err    bool
	}{
		{
			cty.StringVal("tada"),
			cty.StringVal("ce47d07243bb6eaf5e1322c81baf9bbf"),
			false,
		},
		{ // Confirm that we're not trimming any whitespaces
			cty.StringVal(" tada "),
			cty.StringVal("aadf191a583e53062de2d02c008141c4"),
			false,
		},
		{ // We accept empty string too
			cty.StringVal(""),
			cty.StringVal("d41d8cd98f00b204e9800998ecf8427e"),
			false,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("md5(%#v)", test.String), func(t *testing.T) {
			got, err := Md5Func.Call([]cty.Value{test.String})

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

func TestFileMD5(t *testing.T) {
	tests := []struct {
		Path cty.Value
		Want cty.Value
		Err  bool
	}{
		{
			cty.StringVal("testdata/hello.txt"),
			cty.StringVal("b10a8db164e0754105b7a99be72e3fe5"),
			false,
		},
		{
			cty.StringVal("testdata/icon.png"),
			cty.StringVal("d7e6c283185a1078c58213beadca98b0"),
			false,
		},
		{
			cty.StringVal("testdata/missing"),
			cty.NilVal,
			true, // no file exists
		},
	}

	fileMD5 := MakeFileMd5Func(".", noopWrapper)

	for _, test := range tests {
		t.Run(fmt.Sprintf("filemd5(%#v)", test.Path), func(t *testing.T) {
			got, err := fileMD5.Call([]cty.Value{test.Path})

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

func TestSha1(t *testing.T) {
	tests := []struct {
		String cty.Value
		Want   cty.Value
		Err    bool
	}{
		{
			cty.StringVal("test"),
			cty.StringVal("a94a8fe5ccb19ba61c4c0873d391e987982fbbd3"),
			false,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("sha1(%#v)", test.String), func(t *testing.T) {
			got, err := Sha1Func.Call([]cty.Value{test.String})

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

func TestFileSHA1(t *testing.T) {
	tests := []struct {
		Path cty.Value
		Want cty.Value
		Err  bool
	}{
		{
			cty.StringVal("testdata/hello.txt"),
			cty.StringVal("0a4d55a8d778e5022fab701977c5d840bbc486d0"),
			false,
		},
		{
			cty.StringVal("testdata/icon.png"),
			cty.StringVal("2821bcc8379e1bd6f4f31b1e6a1fbb204b4a8be8"),
			false,
		},
		{
			cty.StringVal("testdata/missing"),
			cty.NilVal,
			true, // no file exists
		},
	}

	fileSHA1 := MakeFileSha1Func(".", noopWrapper)

	for _, test := range tests {
		t.Run(fmt.Sprintf("filesha1(%#v)", test.Path), func(t *testing.T) {
			got, err := fileSHA1.Call([]cty.Value{test.Path})

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
