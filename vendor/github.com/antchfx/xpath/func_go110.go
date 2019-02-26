// +build go1.10

package xpath

import "math"

func round(f float64) int {
	return int(math.Round(f))
}
