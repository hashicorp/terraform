// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/getproviders/providerreqs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestParseProviderVersionOverride(t *testing.T) {
	tcs := map[string]struct {
		raw       string
		wantAddr  addrs.Provider
		wantVer   string
		wantError string
	}{
		"namespace and version": {
			raw:      "hashicorp/aws=5.70.0",
			wantAddr: addrs.NewDefaultProvider("aws"),
			wantVer:  "5.70.0",
		},
		"fully qualified source": {
			raw:      "registry.terraform.io/hashicorp/aws=>= 5.0.0",
			wantAddr: addrs.NewDefaultProvider("aws"),
			wantVer:  ">= 5.0.0",
		},
		"constraint with spaces": {
			raw:      "hashicorp/random= ~> 3.6",
			wantAddr: addrs.NewDefaultProvider("random"),
			wantVer:  "~> 3.6",
		},
		"missing separator": {
			raw:       "hashicorp/aws",
			wantError: "The -override-provider option expects SOURCE=VERSION",
		},
		"missing version": {
			raw:       "hashicorp/aws=",
			wantError: "The -override-provider option expects SOURCE=VERSION",
		},
		"invalid version": {
			raw:       "hashicorp/aws=not-a-version",
			wantError: "The version constraint \"not-a-version\"",
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseProviderVersionOverride(tc.raw)
			if tc.wantError != "" {
				if !diags.HasErrors() {
					t.Fatalf("expected error containing %q, got none", tc.wantError)
				}
				if !overrideDiagContains(diags, tc.wantError) {
					t.Fatalf("expected error containing %q, got %#v", tc.wantError, diags)
				}
				return
			}
			if diags.HasErrors() {
				t.Fatalf("unexpected diagnostics: %s", diags.Err())
			}
			if got.Addr != tc.wantAddr {
				t.Fatalf("wrong address\ngot:  %s\nwant: %s", got.Addr, tc.wantAddr)
			}
			if got.Version != tc.wantVer {
				t.Fatalf("wrong version\ngot:  %s\nwant: %s", got.Version, tc.wantVer)
			}
			gotConstraint := providerreqs.VersionConstraintsString(got.Constraint)
			wantConstraint := providerreqs.VersionConstraintsString(providerreqs.MustParseVersionConstraints(tc.wantVer))
			if gotConstraint != wantConstraint {
				t.Fatalf("wrong constraint\ngot:  %s\nwant: %s", gotConstraint, wantConstraint)
			}
		})
	}
}

func TestParseProviderVersionOverrides_lastWins(t *testing.T) {
	got, diags := ParseProviderVersionOverrides([]string{
		"hashicorp/aws=4.67.0",
		"hashicorp/random=3.6.0",
		"hashicorp/aws=5.70.0",
	})
	if diags.HasErrors() {
		t.Fatalf("unexpected diagnostics: %s", diags.Err())
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 overrides, got %d", len(got))
	}

	byAddr := map[addrs.Provider]ProviderVersionOverride{}
	for _, override := range got {
		byAddr[override.Addr] = override
	}

	aws := byAddr[addrs.NewDefaultProvider("aws")]
	if aws.Version != "5.70.0" {
		t.Fatalf("expected last aws override to win, got %q", aws.Version)
	}
	if byAddr[addrs.NewDefaultProvider("random")].Version != "3.6.0" {
		t.Fatalf("expected random override to be preserved")
	}
}

func overrideDiagContains(diags tfdiags.Diagnostics, substr string) bool {
	for _, diag := range diags {
		desc := diag.Description()
		if strings.Contains(desc.Detail, substr) || strings.Contains(desc.Summary, substr) {
			return true
		}
	}
	return false
}
