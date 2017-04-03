package ibmcloud

import (
	"fmt"
	"regexp"
)

func validateVlanName(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	nameLen := len(value)
	if nameLen > 20 {
		errors = append(errors, fmt.Errorf("The vlan name can't exceed 20 characters. The given name %s has %d characters", value, nameLen))
	}
	return
}

func validateSubnetSize(v interface{}, k string) (ws []string, errors []error) {
	value := v.(int)
	validSubnetSize := map[int]bool{
		8:  true,
		16: true,
		32: true,
		64: true,
	}

	if !validSubnetSize[value] {
		errors = append(errors, fmt.Errorf("%q is %d. Permissible values are 8, 16, 32, 64", k, value))
	}
	return
}

func validateRouterHostname(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	validRouterHostnameRegex := regexp.MustCompile(`^(fcr|bcr)`)
	if !validRouterHostnameRegex.MatchString(value) {
		errors = append(errors, fmt.Errorf("%q must start with fcr or bcr based on whether is public/private vlan respectively ", k))
	}
	return
}
