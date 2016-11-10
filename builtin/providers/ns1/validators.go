package ns1

import (
	"fmt"
	"net"
)

func validateJob(v interface{}, k string) (ws []string, es []error) {
	t := v.(string)

	validTypes := map[string]bool{
		"http": true,
		"tcp":  true,
		"dns":  true,
		"ping": true,
	}

	if !validTypes[t] {
		es = append(es, fmt.Errorf(
			"%q contains an invalid type %q. Valid types are %q, %q, %q, %q.",
			k, t, "http", "tcp", "dns", "ping"))
	}

	return
}

func validateNotifyRepeat(v interface{}, k string) (ws []string, es []error) {
	r := v.(int)

	if r != 0 && r < 60 {
		es = append(es, fmt.Errorf(
			"%q must be either 0 or greater than 59. Got %d.", k, r))
	}

	return
}

func validatePolicy(v interface{}, k string) (ws []string, es []error) {
	t := v.(string)

	validPolicies := map[string]bool{
		"all":    true,
		"one":    true,
		"quorum": true,
	}

	if !validPolicies[t] {
		es = append(es, fmt.Errorf(
			"%q contains an invalid policy %q. Valid policies are %q, %q, %q.",
			k, t, "all", "one", "quorum"))
	}

	return
}

// Metadata Validators

func validatePositiveInt(v interface{}, k string) (ws []string, es []error) {
	value := v.(int)
	if value < 0 {
		es = append(es, fmt.Errorf(
			"value must be postive in %q", k))
	}
	return
}

func validatePositiveFloat(v interface{}, k string) (ws []string, es []error) {
	value := v.(float64)
	if value < 0 {
		es = append(es, fmt.Errorf(
			"value must be postive in %q", k))
	}
	return
}

func validateCoordinate(v interface{}, k string) (ws []string, es []error) {
	value := v.(float64)
	if value < -180 || 180 < value {
		es = append(es, fmt.Errorf(
			"value must be in range -180 to 180 in %q", k))
	}
	return
}

func validateGeoregion(v interface{}, k string) (ws []string, es []error) {
	r := v.(string)

	validRegions := map[string]bool{
		"US-EAST":       true,
		"US-CENTRAL":    true,
		"US-WEST":       true,
		"EUROPE":        true,
		"ASIAPAC":       true,
		"SOUTH-AMERICA": true,
		"AFRICA":        true,
	}

	if !validRegions[r] {
		es = append(es, fmt.Errorf(
			"%q contains an invalid region %q. Valid regions are %q, %q, %q, %q, %q, %q, %q.",
			k, r, "US-EAST", "US-CENTRAL", "US-WEST", "EUROPE", "ASIAPAC", "SOUTH-AMERICA", "AFRICA"))
	}
	return
}

func validateCountry(v interface{}, k string) (ws []string, es []error) {
	c := v.(string)

	if len(c) != 2 {
		es = append(es, fmt.Errorf(
			"%q contains an invalid ISO 3166 Country Code %q", k, c))

	}
	return
}

func validateUSState(v interface{}, k string) (ws []string, es []error) {
	s := v.(string)

	if len(s) != 2 {
		es = append(es, fmt.Errorf(
			"%q contains an invalid 2-Character State Code %q", k, s))

	}
	return
}

func validateCAProvince(v interface{}, k string) (ws []string, es []error) {
	p := v.(string)

	if len(p) != 2 {
		es = append(es, fmt.Errorf(
			"%q contains an invalid 2-Character Province Code %q", k, p))

	}
	return
}

func validateNote(v interface{}, k string) (ws []string, es []error) {
	n := v.(string)

	if len(n) >= 256 {
		es = append(es, fmt.Errorf(
			"%q contains a value with more than 256 characters", k))

	}
	return
}

func validateIPPrefix(v interface{}, k string) (ws []string, es []error) {
	p := v.(string)

	if _, _, err := net.ParseCIDR(p); err != nil {
		es = append(es, fmt.Errorf(
			"%q contains an invalid IP Prefix %q", k, p))
	}

	return
}
