// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package oci

import (
	"fmt"
	"github.com/hashicorp/terraform/internal/backend/backendbase"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
	"os"
	"os/user"
	"path"
	"regexp"
	"strings"
	"time"
)

func getEnvSettingWithBlankDefault(s string) string {
	return getEnvSettingWithDefault(s, "")
}

func getEnvSettingWithDefault(s string, dv string) string {
	v := os.Getenv(TfEnvPrefix + s)
	if v != "" {
		return v
	}
	v = os.Getenv(OciEnvPrefix + s)
	if v != "" {
		return v
	}
	v = os.Getenv(s)
	if v != "" {
		return v
	}
	return dv
}
func getDurationFromEnvVar(varName string, defaultValue time.Duration) time.Duration {
	valueStr := getEnvSettingWithDefault(varName, fmt.Sprint(defaultValue))
	duration, err := time.ParseDuration(valueStr)
	if err != nil {
		logger.Error("ERROR while parsing env variable %s value: %v", varName, err)
		return defaultValue
	}
	return duration
}
func getHomeFolder() string {
	if os.Getenv("TF_HOME_OVERRIDE") != "" {
		return os.Getenv("TF_HOME_OVERRIDE")
	}
	current, e := user.Current()
	if e != nil {
		//Give up and try to return something sensible
		home := os.Getenv("HOME")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		return home
	}
	return current.HomeDir
}
func checkProfile(profile string, path string) (err error) {
	var profileRegex = regexp.MustCompile(`^\[(.*)\]`)
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(data)
	splitContent := strings.Split(content, "\n")
	for _, line := range splitContent {
		if match := profileRegex.FindStringSubmatch(line); match != nil && len(match) > 1 && match[1] == profile {
			return nil
		}
	}

	return fmt.Errorf("configuration file did not contain profile: %s", profile)
}

// cleans and expands the path if it contains a tilde , returns the expanded path or the input path as is if not expansion
// was performed
func expandPath(filepath string) string {
	if strings.HasPrefix(filepath, fmt.Sprintf("~%c", os.PathSeparator)) {
		filepath = path.Join(getHomeFolder(), filepath[2:])
	}
	return path.Clean(filepath)
}

func getBackendAttrWithDefault(obj cty.Value, attrName, def string) (cty.Value, bool) {
	value := backendbase.GetAttrDefault(obj, attrName, cty.StringVal(getEnvSettingWithDefault(attrName, def)))
	return value, value.IsKnown() && !value.IsNull()
}

func getBackendAttr(obj cty.Value, attrName string) (cty.Value, bool) {
	return getBackendAttrWithDefault(obj, attrName, "")
}
func uniqueStrings(input []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, val := range input {
		if !seen[val] {
			seen[val] = true
			result = append(result, val)
		}
	}
	return result
}

func validateStringObjectPath(val string, path cty.Path, diags *tfdiags.Diagnostics) {
	if strings.HasPrefix(val, "/") || strings.HasSuffix(val, "/") || strings.Contains(val, "//") {
		*diags = diags.Append(tfdiags.AttributeValue(tfdiags.Error,
			"Invalid Value",
			`The value must not start or end with "/" and also not contain consecutive "/"`,
			path.Copy(),
		))
	}
}
