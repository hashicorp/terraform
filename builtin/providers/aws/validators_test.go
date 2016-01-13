package aws

import (
	"testing"
)

func TestValidateEcrRepositoryName(t *testing.T) {
	validNames := []string{
		"nginx-web-app",
		"project-a/nginx-web-app",
		"domain.ltd/nginx-web-app",
		"3chosome-thing.com/01different-pattern",
		"0123456789/999999999",
		"double/forward/slash",
		"000000000000000",
	}
	for _, v := range validNames {
		_, errors := validateEcrRepositoryName(v, "name")
		if len(errors) != 0 {
			t.Fatalf("%q should be a valid ECR repository name: %q", v, errors)
		}
	}

	invalidNames := []string{
		// length > 256
		"3cho_some-thing.com/01different.-_pattern01different.-_pattern01diff" +
			"erent.-_pattern01different.-_pattern01different.-_pattern01different" +
			".-_pattern01different.-_pattern01different.-_pattern01different.-_pa" +
			"ttern01different.-_pattern01different.-_pattern234567",
		// length < 2
		"i",
		"special@character",
		"different+special=character",
		"double//slash",
		"double..dot",
		"/slash-at-the-beginning",
		"slash-at-the-end/",
	}
	for _, v := range invalidNames {
		_, errors := validateEcrRepositoryName(v, "name")
		if len(errors) == 0 {
			t.Fatalf("%q should be an invalid ECR repository name", v)
		}
	}
}
