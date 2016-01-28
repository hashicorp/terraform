package vault

import (
	"fmt"
	"time"
)

// ValidateDurationString is a ValidateFunc implementation that
// verifies the provided string is a valid parseable duration.
func ValidateDurationString(v interface{}, k string) (ws []string, es []error) {
	_, err := time.ParseDuration(v.(string))
	if err != nil {
		es = append(es, fmt.Errorf("%s: error parsing as duration: %s", k, err))
	}
	return
}

// NormalizeDurationString takes a valid duration string and returns it as a
// normalized string as returned by time.Durations' String() function.
func NormalizeDurationString(v interface{}) string {
	d, err := time.ParseDuration(v.(string))
	if err != nil {
		panic(fmt.Sprintf(
			"Duration string should already be valid by now, but got err: %s", err))
	}

	return d.String()
}
