package funcs

import (
	"fmt"
	"testing"
	"time"

	"github.com/zclconf/go-cty/cty"
)

func TestTimestamp(t *testing.T) {
	currentTime := time.Now().UTC()
	result, err := Timestamp()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	resultTime, err := time.Parse(time.RFC3339, result.AsString())
	if err != nil {
		t.Fatalf("Error parsing timestamp: %s", err)
	}

	if resultTime.Sub(currentTime).Seconds() > 10.0 {
		t.Fatalf("Timestamp Diff too large. Expected: %s\nReceived: %s", currentTime.Format(time.RFC3339), result.AsString())
	}

}

func TestTimeadd(t *testing.T) {
	tests := []struct {
		Time     cty.Value
		Duration cty.Value
		Want     cty.Value
		Err      bool
	}{
		{
			cty.StringVal("2017-11-22T00:00:00Z"),
			cty.StringVal("1s"),
			cty.StringVal("2017-11-22T00:00:01Z"),
			false,
		},
		{
			cty.StringVal("2017-11-22T00:00:00Z"),
			cty.StringVal("10m1s"),
			cty.StringVal("2017-11-22T00:10:01Z"),
			false,
		},
		{ // also support subtraction
			cty.StringVal("2017-11-22T00:00:00Z"),
			cty.StringVal("-1h"),
			cty.StringVal("2017-11-21T23:00:00Z"),
			false,
		},
		{ // Invalid format timestamp
			cty.StringVal("2017-11-22"),
			cty.StringVal("-1h"),
			cty.UnknownVal(cty.String),
			true,
		},
		{ // Invalid format duration (day is not supported by ParseDuration)
			cty.StringVal("2017-11-22T00:00:00Z"),
			cty.StringVal("1d"),
			cty.UnknownVal(cty.String),
			true,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("TimeAdd(%#v, %#v)", test.Time, test.Duration), func(t *testing.T) {
			got, err := TimeAdd(test.Time, test.Duration)

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
