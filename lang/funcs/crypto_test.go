package funcs

import (
	"fmt"
	"testing"

	"github.com/zclconf/go-cty/cty"
	"golang.org/x/crypto/bcrypt"
)

func TestUUID(t *testing.T) {
	result, err := UUID()
	if err != nil {
		t.Fatal(err)
	}

	resultStr := result.AsString()
	if got, want := len(resultStr), 36; got != want {
		t.Errorf("wrong result length %d; want %d", got, want)
	}
}

func TestBase64Sha256(t *testing.T) {
	tests := []struct {
		String cty.Value
		Want   cty.Value
		Err    bool
	}{
		{
			cty.StringVal("test"),
			cty.StringVal("n4bQgYhMfWWaL+qgxVrQFaO/TxsrC4Is0V1sFbDwCgg="),
			false,
		},
		// This would differ because we're base64-encoding hex represantiation, not raw bytes.
		// base64encode(sha256("test")) =
		// "OWY4NmQwODE4ODRjN2Q2NTlhMmZlYWEwYzU1YWQwMTVhM2JmNGYxYjJiMGI4MjJjZDE1ZDZjMTViMGYwMGEwOA=="
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("base64sha256(%#v)", test.String), func(t *testing.T) {
			got, err := Base64Sha256(test.String)

			if test.Err {
				if err == nil {
					t.Fatal("succeeded; want error")
				}
				return
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %s", err)
				}
			}

			if !got.RawEquals(test.Want) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.Want)
			}
		})
	}
}

func TestBase64Sha512(t *testing.T) {
	tests := []struct {
		String cty.Value
		Want   cty.Value
		Err    bool
	}{
		{
			cty.StringVal("test"),
			cty.StringVal("7iaw3Ur350mqGo7jwQrpkj9hiYB3Lkc/iBml1JQODbJ6wYX4oOHV+E+IvIh/1nsUNzLDBMxfqa2Ob1f1ACio/w=="),
			false,
		},
		// This would differ because we're base64-encoding hex represantiation, not raw bytes
		// base64encode(sha512("test")) =
		// "OZWUyNmIwZGQ0YWY3ZTc0OWFhMWE4ZWUzYzEwYWU5OTIzZjYxODk4MDc3MmU0NzNmODgxOWE1ZDQ5NDBlMGRiMjdhYzE4NWY4YTBlMWQ1Zjg0Zjg4YmM4ODdmZDY3YjE0MzczMmMzMDRjYzVmYTlhZDhlNmY1N2Y1MDAyOGE4ZmY="
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("base64sha512(%#v)", test.String), func(t *testing.T) {
			got, err := Base64Sha512(test.String)

			if test.Err {
				if err == nil {
					t.Fatal("succeeded; want error")
				}
				return
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %s", err)
				}
			}

			if !got.RawEquals(test.Want) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.Want)
			}
		})
	}
}

func TestBcrypt(t *testing.T) {
	// single variable test
	p, err := Bcrypt(cty.StringVal("test"))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(p.AsString()), []byte("test"))
	if err != nil {
		t.Fatalf("Error comparing hash and password: %s", err)
	}

	// testing with two parameters
	p, err = Bcrypt(cty.StringVal("test"), cty.NumberIntVal(5))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(p.AsString()), []byte("test"))
	if err != nil {
		t.Fatalf("Error comparing hash and password: %s", err)
	}

	// Negative test for more than two parameters
	_, err = Bcrypt(cty.StringVal("test"), cty.NumberIntVal(10), cty.NumberIntVal(11))
	if err == nil {
		t.Fatal("succeeded; want error")
	}
}
