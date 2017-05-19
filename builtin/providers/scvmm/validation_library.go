package scvmm

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/pborman/uuid"
)

func validateTimeout(v interface{}, k string) (warnings []string, errors []error) {
	timeoutString := v.(string)
	_, err := strconv.ParseInt(timeoutString, 10, 64)
	if err != nil {
		return returnError("Timeout should be a natural number with base 10", err)
	}
	return nil, nil
}

func validateVMName(v interface{}, k string) (warnings []string, errors []error) {
	vmName := v.(string)
	match, err := regexp.MatchString("^[^*?:<>|/[\\]]+$", vmName)
	if err != nil {
		return returnError("Regex error: Please report bug to terraform", err)
	}
	if !match {
		return returnError("VM Name should not contain special chars (^*?:<>|/\\).", fmt.Errorf("VM Name is not correct"))
	}
	return nil, nil
}

func validateTemplateName(v interface{}, k string) (warnings []string, errors []error) {
	vmName := v.(string)
	match, err := regexp.MatchString("^[^*?:<>|/[\\]]+$", vmName)
	if err != nil {
		return returnError("Regex error: Please report bug to terraform", err)
	}
	if !match {
		return returnError("Template Name should not contain special chars (^*?:<>|/\\).", fmt.Errorf("Template Name is not correct"))
	}
	return nil, nil
}

func validateVMMServer(v interface{}, k string) (warnings []string, errors []error) {
	vmName := v.(string)
	match, err := regexp.MatchString("^[\\w\\d.\\-_]+$", vmName)
	if err != nil {
		return returnError("Regex error: Please report bug to terraform", err)
	}
	if !match {
		return returnError("Name should not contain special chars (^*?:<>|/\\).", fmt.Errorf("VMMServer Name is not correct"))
	}
	return nil, nil
}

func validateCloudName(v interface{}, k string) (warnings []string, errors []error) {
	vmName := v.(string)
	match, err := regexp.MatchString("^[\\w\\s\\d.\\-_]+$", vmName)
	if err != nil {
		return returnError("Regex error: Please report bug to terraform", err)
	}
	if !match {
		return returnError("Cloud Name should not contain special chars (^*?:<>|/\\).", fmt.Errorf("Cloud Name Name is not correct"))
	}
	return nil, nil
}

func validateGUID(v interface{}, k string) (warnings []string, errors []error) {
	input := v.(string)
	uuid := uuid.Parse(input)
	if uuid == nil {
		return returnError("GUID is not correct, it should be in format xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx", fmt.Errorf("GUID is not correct"))
	}
	return nil, nil
}

func returnError(message string, err error) (warnings []string, errors []error) {
	var errorVar []error
	var warningVar []string
	return append(warningVar, message), append(errorVar, err)
}
