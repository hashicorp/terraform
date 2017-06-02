package kubernetes

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"
	apiValidation "k8s.io/apimachinery/pkg/api/validation"
	utilValidation "k8s.io/apimachinery/pkg/util/validation"
)

func validateAnnotations(value interface{}, key string) (ws []string, es []error) {
	m := value.(map[string]interface{})
	for k, _ := range m {
		errors := utilValidation.IsQualifiedName(strings.ToLower(k))
		if len(errors) > 0 {
			for _, e := range errors {
				es = append(es, fmt.Errorf("%s (%q) %s", key, k, e))
			}
		}
	}
	return
}

func validateName(value interface{}, key string) (ws []string, es []error) {
	v := value.(string)

	errors := apiValidation.NameIsDNSLabel(v, false)
	if len(errors) > 0 {
		for _, err := range errors {
			es = append(es, fmt.Errorf("%s %s", key, err))
		}
	}
	return
}

func validateGenerateName(value interface{}, key string) (ws []string, es []error) {
	v := value.(string)

	errors := apiValidation.NameIsDNSLabel(v, true)
	if len(errors) > 0 {
		for _, err := range errors {
			es = append(es, fmt.Errorf("%s %s", key, err))
		}
	}
	return
}

func validateLabels(value interface{}, key string) (ws []string, es []error) {
	m := value.(map[string]interface{})
	for k, v := range m {
		for _, msg := range utilValidation.IsQualifiedName(k) {
			es = append(es, fmt.Errorf("%s (%q) %s", key, k, msg))
		}
		val := v.(string)
		for _, msg := range utilValidation.IsValidLabelValue(val) {
			es = append(es, fmt.Errorf("%s (%q) %s", key, val, msg))
		}
	}
	return
}

func validateResourceList(value interface{}, key string) (ws []string, es []error) {
	m := value.(map[string]interface{})
	for k, value := range m {
		if _, ok := value.(int); ok {
			continue
		}

		if v, ok := value.(string); ok {
			_, err := resource.ParseQuantity(v)
			if err != nil {
				es = append(es, fmt.Errorf("%s.%s (%q): %s", key, k, v, err))
			}
			continue
		}

		err := "Value can be either string or int"
		es = append(es, fmt.Errorf("%s.%s (%#v): %s", key, k, value, err))
	}
	return
}
