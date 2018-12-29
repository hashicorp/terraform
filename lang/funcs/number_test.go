package funcs

import (
	"fmt"
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestCeil(t *testing.T) {
	tests := []struct {
		Num  cty.Value
		Want cty.Value
		Err  bool
	}{
		{
			cty.NumberFloatVal(-1.8),
			cty.NumberFloatVal(-1),
			false,
		},
		{
			cty.NumberFloatVal(1.2),
			cty.NumberFloatVal(2),
			false,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("ceil(%#v)", test.Num), func(t *testing.T) {
			got, err := Ceil(test.Num)

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

func TestFloor(t *testing.T) {
	tests := []struct {
		Num  cty.Value
		Want cty.Value
		Err  bool
	}{
		{
			cty.NumberFloatVal(-1.8),
			cty.NumberFloatVal(-2),
			false,
		},
		{
			cty.NumberFloatVal(1.2),
			cty.NumberFloatVal(1),
			false,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("floor(%#v)", test.Num), func(t *testing.T) {
			got, err := Floor(test.Num)

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

func TestLog(t *testing.T) {
	tests := []struct {
		Num  cty.Value
		Base cty.Value
		Want cty.Value
		Err  bool
	}{
		{
			cty.NumberFloatVal(1),
			cty.NumberFloatVal(10),
			cty.NumberFloatVal(0),
			false,
		},
		{
			cty.NumberFloatVal(10),
			cty.NumberFloatVal(10),
			cty.NumberFloatVal(1),
			false,
		},

		{
			cty.NumberFloatVal(0),
			cty.NumberFloatVal(10),
			cty.NegativeInfinity,
			false,
		},
		{
			cty.NumberFloatVal(10),
			cty.NumberFloatVal(0),
			cty.NumberFloatVal(-0),
			false,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("log(%#v, %#v)", test.Num, test.Base), func(t *testing.T) {
			got, err := Log(test.Num, test.Base)

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

func TestPow(t *testing.T) {
	tests := []struct {
		Num   cty.Value
		Power cty.Value
		Want  cty.Value
		Err   bool
	}{
		{
			cty.NumberFloatVal(1),
			cty.NumberFloatVal(0),
			cty.NumberFloatVal(1),
			false,
		},
		{
			cty.NumberFloatVal(1),
			cty.NumberFloatVal(1),
			cty.NumberFloatVal(1),
			false,
		},

		{
			cty.NumberFloatVal(2),
			cty.NumberFloatVal(0),
			cty.NumberFloatVal(1),
			false,
		},
		{
			cty.NumberFloatVal(2),
			cty.NumberFloatVal(1),
			cty.NumberFloatVal(2),
			false,
		},
		{
			cty.NumberFloatVal(3),
			cty.NumberFloatVal(2),
			cty.NumberFloatVal(9),
			false,
		},
		{
			cty.NumberFloatVal(-3),
			cty.NumberFloatVal(2),
			cty.NumberFloatVal(9),
			false,
		},
		{
			cty.NumberFloatVal(2),
			cty.NumberFloatVal(-2),
			cty.NumberFloatVal(0.25),
			false,
		},
		{
			cty.NumberFloatVal(0),
			cty.NumberFloatVal(2),
			cty.NumberFloatVal(0),
			false,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("pow(%#v, %#v)", test.Num, test.Power), func(t *testing.T) {
			got, err := Pow(test.Num, test.Power)

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

func TestSignum(t *testing.T) {
	tests := []struct {
		Num  cty.Value
		Want cty.Value
		Err  bool
	}{
		{
			cty.NumberFloatVal(0),
			cty.NumberFloatVal(0),
			false,
		},
		{
			cty.NumberFloatVal(12),
			cty.NumberFloatVal(1),
			false,
		},
		{
			cty.NumberFloatVal(-29),
			cty.NumberFloatVal(-1),
			false,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("signum(%#v)", test.Num), func(t *testing.T) {
			got, err := Signum(test.Num)

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
