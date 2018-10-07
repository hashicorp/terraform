package types

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	homedir "github.com/mitchellh/go-homedir"
)

var (
	becomeMethods = map[string]bool{
		"sudo":   true,
		"su":     true,
		"pbrun":  true,
		"pfexec": true,
		"doas":   true,
		"dzdo":   true,
		"ksu":    true,
		"runas":  true,
	}
)

func vfBecomeMethod(val interface{}, key string) (warns []string, errs []error) {
	v := val.(string)
	if !becomeMethods[v] {
		errs = append(errs, fmt.Errorf("%s is not a valid become_method", v))
	}
	return
}

func vfPath(val interface{}, key string) (warns []string, errs []error) {
	v := val.(string)
	if strings.Index(v, "${path.module}") > -1 {
		warns = append(warns, fmt.Sprintf("I could not reliably determine the existence of '%s', most likely because of https://github.com/hashicorp/terraform/issues/17439. If the file does not exist, you'll experience a failure at runtime.", v))
	} else {
		if _, err := ResolvePath(v); err != nil {
			errs = append(errs, fmt.Errorf("file '%s' does not exist", v))
		}
	}
	return
}

// VfPathDirectory validates existence of a path and that the path is a directory.
func VfPathDirectory(val interface{}, key string) (warns []string, errs []error) {
	v := val.(string)
	if strings.Index(v, "${path.module}") > -1 {
		warns = append(warns, fmt.Sprintf("I could not reliably determine the existence of '%s', most likely because of https://github.com/hashicorp/terraform/issues/17439. If the file does not exist, you'll experience a failure at runtime.", v))
	} else {
		if _, err := ResolveDirectory(v); err != nil {
			errs = append(errs, fmt.Errorf("directory '%s' does not exist or path not directory", v))
		}
	}
	return
}

func mapFromTypeMap(v interface{}) map[string]interface{} {
	switch v := v.(type) {
	case nil:
		return make(map[string]interface{})
	case map[string]interface{}:
		return v
	default:
		panic(fmt.Sprintf("Unsupported type: %T", v))
	}
}

func mapFromTypeSet(i interface{}) map[string]interface{} {
	return i.(map[string]interface{})
}

func mapFromTypeSetList(i []interface{}) map[string]interface{} {
	for _, v := range i {
		return mapFromTypeSet(v)
	}
	return make(map[string]interface{})
}

func listOfInterfaceToListOfString(v interface{}) []string {
	var result []string
	switch v := v.(type) {
	case nil:
		return result
	case []interface{}:
		for _, vv := range v {
			if vv, ok := vv.(string); ok {
				result = append(result, vv)
			}
		}
		return result
	default:
		panic(fmt.Sprintf("Unsupported type: %T", v))
	}
}

// ResolvePath checks if a path exists.
func ResolvePath(path string) (string, error) {
	expandedPath, _ := homedir.Expand(path)
	if _, err := os.Stat(expandedPath); err == nil {
		return filepath.Clean(expandedPath), nil
	}
	return "", fmt.Errorf("Ansible module not found at path: [%s]", path)
}

// ResolveDirectory checks if a path exists and is a directory.
func ResolveDirectory(path string) (string, error) {
	expandedPath, _ := homedir.Expand(path)
	if stat, err := os.Stat(expandedPath); err == nil {
		if !stat.IsDir() {
			return "", fmt.Errorf("Path [%s] must be a directory", path)
		}
		return filepath.Clean(expandedPath), nil
	}
	return "", fmt.Errorf("Ansible module not found at path: [%s]", path)
}
