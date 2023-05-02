// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package httpclient

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/version"
)

func TestUserAgentString_env(t *testing.T) {
	expectedBase := fmt.Sprintf(userAgentFormat, version.Version)
	if oldenv, isSet := os.LookupEnv(uaEnvVar); isSet {
		defer os.Setenv(uaEnvVar, oldenv)
	} else {
		defer os.Unsetenv(uaEnvVar)
	}

	for i, c := range []struct {
		expected   string
		additional string
	}{
		{expectedBase, ""},
		{expectedBase, " "},
		{expectedBase, " \n"},

		{fmt.Sprintf("%s test/1", expectedBase), "test/1"},
		{fmt.Sprintf("%s test/2", expectedBase), "test/2 "},
		{fmt.Sprintf("%s test/3", expectedBase), " test/3 "},
		{fmt.Sprintf("%s test/4", expectedBase), "test/4 \n"},
	} {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			if c.additional == "" {
				os.Unsetenv(uaEnvVar)
			} else {
				os.Setenv(uaEnvVar, c.additional)
			}

			actual := UserAgentString()

			if c.expected != actual {
				t.Fatalf("Expected User-Agent '%s' does not match '%s'", c.expected, actual)
			}
		})
	}
}

func TestUserAgentAppendViaEnvVar(t *testing.T) {
	if oldenv, isSet := os.LookupEnv(uaEnvVar); isSet {
		defer os.Setenv(uaEnvVar, oldenv)
	} else {
		defer os.Unsetenv(uaEnvVar)
	}

	expectedBase := "HashiCorp Terraform/0.0.0 (+https://www.terraform.io)"

	testCases := []struct {
		envVarValue string
		expected    string
	}{
		{"", expectedBase},
		{" ", expectedBase},
		{" \n", expectedBase},
		{"test/1", expectedBase + " test/1"},
		{"test/1 (comment)", expectedBase + " test/1 (comment)"},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			os.Unsetenv(uaEnvVar)
			os.Setenv(uaEnvVar, tc.envVarValue)
			givenUA := TerraformUserAgent("0.0.0")
			if givenUA != tc.expected {
				t.Fatalf("Expected User-Agent '%s' does not match '%s'", tc.expected, givenUA)
			}
		})
	}
}
