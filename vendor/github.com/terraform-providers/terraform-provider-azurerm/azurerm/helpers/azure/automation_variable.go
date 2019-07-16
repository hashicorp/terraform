package azure

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

func ParseAzureRmAutomationVariableValue(resource string, input *string) (interface{}, error) {
	if input == nil {
		if resource != "azurerm_automation_variable_null" {
			return nil, fmt.Errorf("Expected value \"nil\" to be %q, actual type is \"azurerm_automation_variable_null\"", resource)
		}
		return nil, nil
	}

	var value interface{}
	var err error
	actualResource := "Unknown"
	datePattern := regexp.MustCompile(`"\\/Date\((-?[0-9]+)\)\\/"`)
	matches := datePattern.FindStringSubmatch(*input)

	if len(matches) == 2 && matches[0] == *input {
		if ticks, err := strconv.ParseInt(matches[1], 10, 64); err == nil {
			value = time.Unix(ticks/1000, ticks%1000*1000000).In(time.UTC)
			actualResource = "azurerm_automation_variable_datetime"
		}
	} else if value, err = strconv.Unquote(*input); err == nil {
		actualResource = "azurerm_automation_variable_string"
	} else if value, err = strconv.ParseBool(*input); err == nil {
		actualResource = "azurerm_automation_variable_bool"
	} else if value, err = strconv.ParseInt(*input, 10, 32); err == nil {
		value = int32(value.(int64))
		actualResource = "azurerm_automation_variable_int"
	}

	if actualResource != resource {
		return nil, fmt.Errorf("Expected value %q to be %q, actual type is %q", *input, resource, actualResource)
	}
	return value, nil
}
