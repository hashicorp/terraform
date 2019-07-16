package validate

import (
	"fmt"
	"regexp"
)

func HDInsightClusterVersion(i interface{}, k string) (warnings []string, errors []error) {
	version := i.(string)

	// 3.6, 3333.6666 or 1.2.3000.45
	// `major minor`
	re := regexp.MustCompile(`^(\d)+(.){1}(\d)+$`)
	if re != nil && !re.MatchString(version) {
		// otherwise retry using `major minor build release`
		re = regexp.MustCompile(`^(\d)+(.)(\d)+(.)(\d)+(.)(\d)+$`)
		if re != nil && !re.MatchString(version) {
			errors = append(errors, fmt.Errorf("%s must be a version in the format `x.y` or `a.b.c.d` - got %q.", k, version))
		}
	}

	return warnings, errors
}

func HDInsightName(v interface{}, k string) (warnings []string, errors []error) {
	value := v.(string)

	// The name must be 59 characters or less and can contain letters, numbers, and hyphens (but the first and last character must be a letter or number).
	if matched := regexp.MustCompile(`(^[a-zA-Z0-9])([a-zA-Z0-9-]{1,57})([a-zA-Z0-9]$)`).Match([]byte(value)); !matched {
		errors = append(errors, fmt.Errorf("%q must be 59 characters or less and can contain letters, numbers, and hyphens (but the first and last character must be a letter or number).", k))
	}

	return warnings, errors
}
