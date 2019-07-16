package validate

import (
	"fmt"
	"regexp"

	"github.com/Azure/azure-sdk-for-go/services/devtestlabs/mgmt/2016-05-15/dtl"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func DevTestLabName() schema.SchemaValidateFunc {
	return validation.StringMatch(
		regexp.MustCompile("^[A-Za-z0-9_-]+$"),
		"Lab Name can only include alphanumeric characters, underscores, hyphens.")
}

func DevTestVirtualMachineName(maxLength int) schema.SchemaValidateFunc {
	return func(i interface{}, k string) ([]string, []error) {
		v, ok := i.(string)
		if !ok {
			return nil, []error{
				fmt.Errorf("expected type of %s to be string", k),
			}
		}

		errs := make([]error, 0)
		if 1 <= len(v) && len(v) > maxLength {
			errs = append(errs, fmt.Errorf("Expected %s to be between 1 and %d characters - got %d", k, maxLength, len(v)))
		}

		matched, err := regexp.MatchString("^([a-zA-Z0-9]{1})([a-zA-Z0-9-]+)([a-zA-Z0-9]{1})$", v)
		if err != nil {
			errs = append(errs, fmt.Errorf("Error validating regex: %+v", err))
		}
		if !matched {
			errs = append(errs, fmt.Errorf("%s may contain letters, numbers, or '-', must begin and end with a letter or number, and cannot be all numbers.", k))
		}

		matched, err = regexp.MatchString("([a-zA-Z-]+)", v)
		if err != nil {
			errs = append(errs, fmt.Errorf("Error validating regex: %+v", err))
		}
		if !matched {
			errs = append(errs, fmt.Errorf("%s cannot be all numbers.", k))
		}

		return nil, errs
	}
}

func DevTestVirtualNetworkUsagePermissionType() schema.SchemaValidateFunc {
	return validation.StringInSlice([]string{
		string(dtl.Allow),
		string(dtl.Default),
		string(dtl.Deny),
	}, false)
}
