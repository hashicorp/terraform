// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package version

import (
	"regexp"
	"strings"
	"testing"
)

// Smoke test to validate that the version file can be read correctly and all exported
// variables include the expected information.
func TestVersion(t *testing.T) {
	if match, _ := regexp.MatchString("[^\\d+\\.]", Version); match != false {
		t.Fatalf("Version should contain only the main version")
	}

	if match, _ := regexp.MatchString("[^a-z\\d]", Prerelease); match != false {
		t.Fatalf("Prerelease should contain only letters and numbers")
	}

	if SemVer.Prerelease() != "" {
		t.Fatalf("SemVer should not include prerelease information")
	}

	if !strings.Contains(String(), Prerelease) {
		t.Fatalf("Full version string should include prerelease information")
	}
}
