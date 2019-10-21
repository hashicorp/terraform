// +build !go1.10

package xpath

import "math"

// math.Round() is supported by Go 1.10+,
// This method just compatible for version <1.10.
// https://github.com/golang/go/issues/20100
func round(f float64) int {
	if math.Abs(f) < 0.5 {
		return 0
	}
	return int(f + math.Copysign(0.5, f))
}
