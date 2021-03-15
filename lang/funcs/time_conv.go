package funcs

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

var inSeconds = map[string]float64{
	"seconds": 1,
	"minutes": 60,
	"hours":   60 * 60,
	"days":    60 * 60 * 24,
	"weeks":   60 * 60 * 24 * 7,
}

type interval struct {
	Magnitude float64
	Unit      string
}

// ValidIntervalUnits returns the intervals that can be used in the to_* time conversion functions.
// It shadows the keys of the inSeconds map, and is largely just a convenience
// Intentionally missing from ValidIntervalUnits are any measures of time which may have a variable length.
// For example, a year could be either 365 or 366 days (or other lengths in other calendars!), and a similar situation
// exists for months. Because of this, we omit them entirely, as this grey area could cause confusion for the user.
func ValidIntervalUnits() []string {
	valids := make([]string, 0, len(inSeconds))
	for k := range inSeconds {
		valids = append(valids, k)
	}

	return valids
}

func validIntervalUnitsWithSingulars() []string {
	valids := ValidIntervalUnits()
	withPlurals := make([]string, 0, len(valids)*2)
	for _, unit := range valids {
		withPlurals = append(withPlurals, strings.TrimSuffix(unit, "s"))
		withPlurals = append(withPlurals, unit)
	}
	return withPlurals
}

func parseInterval(intervalStr, caller string) (interval, error) {
	split := strings.Split(strings.ToLower(intervalStr), " ")
	magnitude, floatParseErr := strconv.ParseFloat(split[0], 64)
	unit := split[1]
	if !strings.HasSuffix(unit, "s") { // If it's not a plural, pluralise it
		unit += "s"
	}
	_, isValidInterval := inSeconds[unit]

	if len(split) != 2 || floatParseErr != nil || !isValidInterval {
		return interval{}, fmt.Errorf("The argument to %s must be a non-zero number and a valid unit, separated by a single space. Valid units are: %v. Singular and plural units are equivalent", caller, validIntervalUnitsWithSingulars())
	}

	return interval{Magnitude: magnitude, Unit: unit}, nil
}

// NewTimeConvFunc returns a cty.Function that will convert a time interval (represented as a string such as "1 day", "127 hours" or "36 minutes")
// into the given unit. NewTimeConvFunc will panic if it's passed a unit that's not in funcs.ValidIntervalUnits
func NewTimeConvFunc(toUnit string) function.Function {
	if _, ok := inSeconds[toUnit]; !ok {
		panic(fmt.Sprintf("The interval passed to NewTimeConvFunc (%s) must be a valid interval. Valid intervals are %v", toUnit, ValidIntervalUnits()))
	}

	return function.New(&function.Spec{
		Params: []function.Parameter{{
			Name: "interval",
			Type: cty.String,
		}},
		Type: function.StaticReturnType(cty.Number),
		Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
			intervalStr := args[0].AsString()

			interval, err := parseInterval(intervalStr, "to_seconds")
			if err != nil {
				return cty.UnknownVal(cty.String), err
			}

			result := (interval.Magnitude * inSeconds[interval.Unit]) / inSeconds[toUnit]
			return cty.NumberFloatVal(result), nil
		},
	})
}
