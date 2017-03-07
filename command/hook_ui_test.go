package command

import (
	"fmt"
	"testing"
)

func TestTruncateId(t *testing.T) {
	testCases := []struct {
		Input    string
		Expected string
		MaxLen   int
	}{
		{
			Input:    "Hello world",
			Expected: "H...d",
			MaxLen:   3,
		},
		{
			Input:    "Hello world",
			Expected: "H...d",
			MaxLen:   5,
		},
		{
			Input:    "Hello world",
			Expected: "He...d",
			MaxLen:   6,
		},
		{
			Input:    "Hello world",
			Expected: "He...ld",
			MaxLen:   7,
		},
		{
			Input:    "Hello world",
			Expected: "Hel...ld",
			MaxLen:   8,
		},
		{
			Input:    "Hello world",
			Expected: "Hel...rld",
			MaxLen:   9,
		},
		{
			Input:    "Hello world",
			Expected: "Hell...rld",
			MaxLen:   10,
		},
		{
			Input:    "Hello world",
			Expected: "Hello world",
			MaxLen:   11,
		},
		{
			Input:    "Hello world",
			Expected: "Hello world",
			MaxLen:   12,
		},
	}
	for i, tc := range testCases {
		testName := fmt.Sprintf("%d", i)
		t.Run(testName, func(t *testing.T) {
			out := truncateId(tc.Input, tc.MaxLen)
			if out != tc.Expected {
				t.Fatalf("Expected %q to be shortened to %d as %q (given: %q)",
					tc.Input, tc.MaxLen, tc.Expected, out)
			}
		})
	}
}
