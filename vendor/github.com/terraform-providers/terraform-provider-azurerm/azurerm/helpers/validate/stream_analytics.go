package validate

import "fmt"

func StreamAnalyticsJobStreamingUnits(i interface{}, k string) (w []string, es []error) {
	v, ok := i.(int)
	if !ok {
		es = append(es, fmt.Errorf("expected type of %s to be int", k))
		return
	}

	//  Property 'streamingUnits' value '5' is not in the acceptable set: '1','3','6','12', and multiples of 6 up to your quota"
	if v == 1 || v == 3 {
		return
	}

	if v < 1 || v > 120 {
		es = append(es, fmt.Errorf("expected %s to be in the range (1 - 120), got %d", k, v))
		return
	}

	if v%6 != 0 {
		es = append(es, fmt.Errorf("expected %s to be divisible by 6, got %d", k, v))
		return
	}

	return
}
