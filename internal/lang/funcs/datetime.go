// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package funcs

import (
	"fmt"
	"time"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

// TimestampFunc constructs a function that returns a string representation of the current date and time.
var TimestampFunc = function.New(&function.Spec{
	Params:       []function.Parameter{},
	Type:         function.StaticReturnType(cty.String),
	RefineResult: refineNotNull,
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		return cty.StringVal(time.Now().UTC().Format(time.RFC3339)), nil
	},
})

// MakeStaticTimestampFunc constructs a function that returns a string
// representation of the date and time specified by the provided argument.
func MakeStaticTimestampFunc(static time.Time) function.Function {
	return function.New(&function.Spec{
		Params: []function.Parameter{},
		Type:   function.StaticReturnType(cty.String),
		Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
			return cty.StringVal(static.Format(time.RFC3339)), nil
		},
	})
}

// TimeAddFunc constructs a function that adds a duration to a timestamp, returning a new timestamp.
var TimeAddFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "timestamp",
			Type: cty.String,
		},
		{
			Name: "duration",
			Type: cty.String,
		},
	},
	Type:         function.StaticReturnType(cty.String),
	RefineResult: refineNotNull,
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		ts, err := parseTimestamp(args[0].AsString())
		if err != nil {
			return cty.UnknownVal(cty.String), err
		}
		duration, err := time.ParseDuration(args[1].AsString())
		if err != nil {
			return cty.UnknownVal(cty.String), err
		}

		return cty.StringVal(ts.Add(duration).Format(time.RFC3339)), nil
	},
})

// TimeCmpFunc is a function that compares two timestamps.
var TimeCmpFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "timestamp_a",
			Type: cty.String,
		},
		{
			Name: "timestamp_b",
			Type: cty.String,
		},
	},
	Type:         function.StaticReturnType(cty.Number),
	RefineResult: refineNotNull,
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		tsA, err := parseTimestamp(args[0].AsString())
		if err != nil {
			return cty.UnknownVal(cty.String), function.NewArgError(0, err)
		}
		tsB, err := parseTimestamp(args[1].AsString())
		if err != nil {
			return cty.UnknownVal(cty.String), function.NewArgError(1, err)
		}

		switch {
		case tsA.Equal(tsB):
			return cty.NumberIntVal(0), nil
		case tsA.Before(tsB):
			return cty.NumberIntVal(-1), nil
		default:
			// By elimintation, tsA must be after tsB.
			return cty.NumberIntVal(1), nil
		}
	},
})

// Timestamp returns a string representation of the current date and time.
//
// In the Terraform language, timestamps are conventionally represented as
// strings using RFC 3339 "Date and Time format" syntax, and so timestamp
// returns a string in this format.
func Timestamp() (cty.Value, error) {
	return TimestampFunc.Call([]cty.Value{})
}

// TimeAdd adds a duration to a timestamp, returning a new timestamp.
//
// In the Terraform language, timestamps are conventionally represented as
// strings using RFC 3339 "Date and Time format" syntax. Timeadd requires
// the timestamp argument to be a string conforming to this syntax.
//
// `duration` is a string representation of a time difference, consisting of
// sequences of number and unit pairs, like `"1.5h"` or `1h30m`. The accepted
// units are `ns`, `us` (or `µs`), `"ms"`, `"s"`, `"m"`, and `"h"`. The first
// number may be negative to indicate a negative duration, like `"-2h5m"`.
//
// The result is a string, also in RFC 3339 format, representing the result
// of adding the given direction to the given timestamp.
func TimeAdd(timestamp cty.Value, duration cty.Value) (cty.Value, error) {
	return TimeAddFunc.Call([]cty.Value{timestamp, duration})
}

// TimeCmp compares two timestamps, indicating whether they are equal or
// if one is before the other.
//
// TimeCmp considers the UTC offset of each given timestamp when making its
// decision, so for example 6:00 +0200 and 4:00 UTC are equal.
//
// In the Terraform language, timestamps are conventionally represented as
// strings using RFC 3339 "Date and Time format" syntax. TimeCmp requires
// the timestamp argument to be a string conforming to this syntax.
//
// The result is always a number between -1 and 1. -1 indicates that
// timestampA is earlier than timestampB. 1 indicates that timestampA is
// later. 0 indicates that the two timestamps represent the same instant.
func TimeCmp(timestampA, timestampB cty.Value) (cty.Value, error) {
	return TimeCmpFunc.Call([]cty.Value{timestampA, timestampB})
}

func parseTimestamp(ts string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		switch err := err.(type) {
		case *time.ParseError:
			// If err is a time.ParseError then its string representation is not
			// appropriate since it relies on details of Go's strange date format
			// representation, which a caller of our functions is not expected
			// to be familiar with.
			//
			// Therefore we do some light transformation to get a more suitable
			// error that should make more sense to our callers. These are
			// still not awesome error messages, but at least they refer to
			// the timestamp portions by name rather than by Go's example
			// values.
			if err.LayoutElem == "" && err.ValueElem == "" && err.Message != "" {
				// For some reason err.Message is populated with a ": " prefix
				// by the time package.
				return time.Time{}, fmt.Errorf("not a valid RFC3339 timestamp%s", err.Message)
			}
			var what string
			switch err.LayoutElem {
			case "2006":
				what = "year"
			case "01":
				what = "month"
			case "02":
				what = "day of month"
			case "15":
				what = "hour"
			case "04":
				what = "minute"
			case "05":
				what = "second"
			case "Z07:00":
				what = "UTC offset"
			case "T":
				return time.Time{}, fmt.Errorf("not a valid RFC3339 timestamp: missing required time introducer 'T'")
			case ":", "-":
				if err.ValueElem == "" {
					return time.Time{}, fmt.Errorf("not a valid RFC3339 timestamp: end of string where %q is expected", err.LayoutElem)
				} else {
					return time.Time{}, fmt.Errorf("not a valid RFC3339 timestamp: found %q where %q is expected", err.ValueElem, err.LayoutElem)
				}
			default:
				// Should never get here, because time.RFC3339 includes only the
				// above portions, but since that might change in future we'll
				// be robust here.
				what = "timestamp segment"
			}
			if err.ValueElem == "" {
				return time.Time{}, fmt.Errorf("not a valid RFC3339 timestamp: end of string before %s", what)
			} else {
				return time.Time{}, fmt.Errorf("not a valid RFC3339 timestamp: cannot use %q as %s", err.ValueElem, what)
			}
		}
		return time.Time{}, err
	}
	return t, nil
}
