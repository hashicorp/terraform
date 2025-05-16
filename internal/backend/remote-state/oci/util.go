// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package oci

import (
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/terraform/internal/backend/backendbase"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

func getEnvSettingWithBlankDefault(s string) string {
	return getEnvSettingWithDefault(s, "")
}

func getEnvSettingWithDefault(s string, dv string) string {
	v := os.Getenv(OciEnvPrefix + s)
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
		loggerFunc().Error("ERROR while parsing env variable %s value: %v", varName, err)
		return defaultValue
	}
	return duration
}
func getHomeFolder() string {
	if os.Getenv("OCI_HOME_OVERRIDE") != "" {
		return os.Getenv("OCI_HOME_OVERRIDE")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		loggerFunc().Error("ERROR while getting home directory: %v", err)
		return ""
	}
	return home
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

func validateStringWorkspacePrefix(val string, path cty.Path, diags *tfdiags.Diagnostics) {
	if strings.HasPrefix(val, "/") || strings.Contains(val, "//") {
		*diags = diags.Append(tfdiags.AttributeValue(tfdiags.Error,
			"Invalid Value",
			`The value must not start  with "/" and also not contain consecutive "/"`,
			path.Copy(),
		))
	}
}

func validateStringBucketName(val string, path cty.Path, diags *tfdiags.Diagnostics) {
	match, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, val)
	if !match {
		*diags = diags.Append(tfdiags.AttributeValue(tfdiags.Error,
			"Invalid Value",
			`The bucket name can only include alphanumeric characters, underscores (_), and hyphens (-).`,
			path.Copy(),
		))
	}
}
func requiredAttributeErrDiag(path cty.Path) tfdiags.Diagnostic {
	return tfdiags.AttributeValue(tfdiags.Error,
		"Missing Required Value",
		fmt.Sprintf("The attribute %q is required by the backend.\n\n", path)+
			"Refer to the backend documentation for additional information which attributes are required.",
		path,
	)
}
func attributeErrDiag(summary, detail string, path cty.Path) tfdiags.Diagnostic {
	return tfdiags.AttributeValue(
		tfdiags.Error,
		summary,
		detail,
		path,
	)
}

func attributeWarningDiag(summary, detail string, path cty.Path) tfdiags.Diagnostic {
	return tfdiags.AttributeValue(
		tfdiags.Warning,
		summary,
		detail,
		path,
	)
}
