package gitlab

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	gitlab "github.com/xanzy/go-gitlab"
)

// copied from ../github/util.go
func validateValueFunc(values []string) schema.SchemaValidateFunc {
	return func(v interface{}, k string) (we []string, errors []error) {
		value := v.(string)
		valid := false
		for _, role := range values {
			if value == role {
				valid = true
				break
			}
		}

		if !valid {
			errors = append(errors, fmt.Errorf("%s is an invalid value for argument %s", value, k))
		}
		return
	}
}

func stringToVisibilityLevel(s string) *gitlab.VisibilityLevelValue {
	lookup := map[string]gitlab.VisibilityLevelValue{
		"private":  gitlab.PrivateVisibility,
		"internal": gitlab.InternalVisibility,
		"public":   gitlab.PublicVisibility,
	}

	value, ok := lookup[s]
	if !ok {
		return nil
	}
	return &value
}

func visibilityLevelToString(v gitlab.VisibilityLevelValue) *string {
	lookup := map[gitlab.VisibilityLevelValue]string{
		gitlab.PrivateVisibility:  "private",
		gitlab.InternalVisibility: "internal",
		gitlab.PublicVisibility:   "public",
	}
	value, ok := lookup[v]
	if !ok {
		return nil
	}
	return &value
}
