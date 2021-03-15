package funcs

import (
	"fmt"
	"testing"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

type testCase struct {
	Interval cty.Value
	Want     cty.Value
	Err      bool
}

func TestToSeconds(t *testing.T) {
	tests := []testCase{
		{
			cty.StringVal("30 seconds"),
			cty.NumberIntVal(30),
			false,
		},
		{
			cty.StringVal("1 minute"),
			cty.NumberIntVal(60),
			false,
		},
		{
			cty.StringVal("1 minutes"),
			cty.NumberIntVal(60),
			false,
		},
		{
			cty.StringVal("1 MiNuTe"),
			cty.NumberIntVal(60),
			false,
		},
		{
			cty.StringVal("1.5 minutes"),
			cty.NumberIntVal(90),
			false,
		},
		{
			cty.StringVal("1 hour"),
			cty.NumberIntVal(3600),
			false,
		},
		{
			cty.StringVal("9 years"),
			cty.UnknownVal(cty.String),
			true,
		},
		{
			cty.StringVal("not a real interval"),
			cty.UnknownVal(cty.String),
			true,
		},
	}
	run(t, "to_seconds", NewTimeConvFunc("seconds"), tests)
}

func TestToMinutes(t *testing.T) {
	tests := []testCase{
		{
			cty.StringVal("30 minute"),
			cty.NumberFloatVal(30),
			false,
		},
		{
			cty.StringVal("60 seconds"),
			cty.NumberFloatVal(1),
			false,
		},
		{
			cty.StringVal("60 SeCoNdS"),
			cty.NumberFloatVal(1),
			false,
		},
		{
			cty.StringVal("1.5 hours"),
			cty.NumberIntVal(90),
			false,
		},
		{
			cty.StringVal("1 day"),
			cty.NumberIntVal(1440),
			false,
		},
		{
			cty.StringVal("9 years"),
			cty.UnknownVal(cty.String),
			true,
		},
		{
			cty.StringVal("not a real interval"),
			cty.UnknownVal(cty.String),
			true,
		},
	}
	run(t, "to_minutes", NewTimeConvFunc("minutes"), tests)
}

func TestToHours(t *testing.T) {
	tests := []testCase{
		{
			cty.StringVal("2 hours"),
			cty.NumberFloatVal(2),
			false,
		},
		{
			cty.StringVal("30 minute"),
			cty.NumberFloatVal(0.5),
			false,
		},
		{
			cty.StringVal("30 minutes"),
			cty.NumberFloatVal(0.5),
			false,
		},
		{
			cty.StringVal("30 MiNuTeS"),
			cty.NumberFloatVal(0.5),
			false,
		},
		{
			cty.StringVal("1.5 days"),
			cty.NumberIntVal(36),
			false,
		},
		{
			cty.StringVal("1 week"),
			cty.NumberIntVal(168),
			false,
		},
		{
			cty.StringVal("9 years"),
			cty.UnknownVal(cty.String),
			true,
		},
		{
			cty.StringVal("not a real interval"),
			cty.UnknownVal(cty.String),
			true,
		},
	}
	run(t, "to_hours", NewTimeConvFunc("hours"), tests)
}

func TestToDays(t *testing.T) {
	tests := []testCase{
		{
			cty.StringVal("7 days"),
			cty.NumberFloatVal(7),
			false,
		},
		{
			cty.StringVal("24 hours"),
			cty.NumberFloatVal(1),
			false,
		},
		{
			cty.StringVal("36 hours"),
			cty.NumberFloatVal(1.5),
			false,
		},
		{
			cty.StringVal("36 HoUrS"),
			cty.NumberFloatVal(1.5),
			false,
		},
		{
			cty.StringVal("1.5 weeks"),
			cty.NumberFloatVal(10.5),
			false,
		},
		{
			cty.StringVal("1 week"),
			cty.NumberIntVal(7),
			false,
		},
		{
			cty.StringVal("9 years"),
			cty.UnknownVal(cty.String),
			true,
		},
		{
			cty.StringVal("not a real interval"),
			cty.UnknownVal(cty.String),
			true,
		},
	}
	run(t, "to_days", NewTimeConvFunc("days"), tests)
}

func TestToWeeks(t *testing.T) {
	tests := []testCase{
		{
			cty.StringVal("1 week"),
			cty.NumberFloatVal(1),
			false,
		},
		{
			cty.StringVal("7 days"),
			cty.NumberFloatVal(1),
			false,
		},
		{
			cty.StringVal("364 days"),
			cty.NumberFloatVal(52),
			false,
		},
		{
			cty.StringVal("9 years"),
			cty.UnknownVal(cty.String),
			true,
		},
		{
			cty.StringVal("not a real interval"),
			cty.UnknownVal(cty.String),
			true,
		},
	}
	run(t, "to_weeks", NewTimeConvFunc("weeks"), tests)
}

func run(t *testing.T, funcName string, callable function.Function, tests []testCase) {
	for _, test := range tests {
		t.Run(fmt.Sprintf("%s(%q)", funcName, test.Interval.AsString()), func(t *testing.T) {
			got, err := callable.Call([]cty.Value{test.Interval})

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
