package funcs

import (
	"time"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

// TimestampFunc constructs a function that returns a string representation of the current date and time.
var TimestampFunc = function.New(&function.Spec{
	Params: []function.Parameter{},
	Type:   function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		return cty.StringVal(time.Now().UTC().Format(time.RFC3339)), nil
	},
})

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
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		ts, err := time.Parse(time.RFC3339, args[0].AsString())
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
