package validate

import (
	"fmt"
	"regexp"
)

func KubernetesAdminUserName(i interface{}, k string) (warnings []string, errors []error) {
	adminUserName := i.(string)

	re := regexp.MustCompile(`^[A-Za-z][-A-Za-z0-9_]*$`)
	if re != nil && !re.MatchString(adminUserName) {
		errors = append(errors, fmt.Errorf("%s must start with alphabet and/or continue with alphanumeric characters, underscores, hyphens. Got %q.", k, adminUserName))
	}

	return warnings, errors
}

func KubernetesAgentPoolName(i interface{}, k string) (warnings []string, errors []error) {
	agentPoolName := i.(string)

	re := regexp.MustCompile(`^[a-z]{1}[a-z0-9]{0,11}$`)
	if re != nil && !re.MatchString(agentPoolName) {
		errors = append(errors, fmt.Errorf("%s must start with a lowercase letter, have max length of 12, and only have characters a-z0-9. Got %q.", k, agentPoolName))
	}

	return warnings, errors
}

func KubernetesDNSPrefix(i interface{}, k string) (warnings []string, errors []error) {
	dnsPrefix := i.(string)

	re := regexp.MustCompile(`^[a-zA-Z][-a-zA-Z0-9]{0,43}[a-zA-Z0-9]$`)
	if re != nil && !re.MatchString(dnsPrefix) {
		errors = append(errors, fmt.Errorf("%s must contain between 2 and 45 characters. The name can contain only letters, numbers, and hyphens. The name must start with a letter and must end with an alphanumeric character.. Got %q.", k, dnsPrefix))
	}

	return warnings, errors
}
