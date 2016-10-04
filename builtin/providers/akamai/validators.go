package akamai

import (
    "fmt"
    "regexp"
)

func validateContractId(v interface{}, k string) (ws []string, errors []error) {
    value := v.(string)

    if !regexp.MustCompile(`^ctr_`).MatchString(value) {
        errors = append(errors, fmt.Errorf("%q must be prefixed by %q", k, "ctr_"))
    }

    if !regexp.MustCompile(`^ctr_[0-9]-[A-Z0-9]{6}$`).MatchString(value) {
        errors = append(errors, fmt.Errorf("%q doesn't comply with restrictions", k))
    }

    return
}

func validateGroupId(v interface{}, k string) (ws []string, errors []error) {
    value := v.(string)

    if !regexp.MustCompile(`^grp_`).MatchString(value) {
        errors = append(errors, fmt.Errorf("%q must be prefixed by %q", k, "grp_"))
    }

    if !regexp.MustCompile(`^grp_[0-9]{5}$`).MatchString(value) {
        errors = append(errors, fmt.Errorf("%q doesn't comply with restrictions", k))
    }

    return
}
