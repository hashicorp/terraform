// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/getproviders/providerreqs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// ProviderVersionOverride is a provider version constraint supplied via the
// -override-provider command-line flag.
type ProviderVersionOverride struct {
	// Raw is the original flag value, preserved for diagnostics and tests.
	Raw string

	// Addr is the fully-qualified provider address.
	Addr addrs.Provider

	// Version is the original constraint string supplied by the user.
	Version string

	// Constraint is the parsed form of Version.
	Constraint providerreqs.VersionConstraints
}

// ParseProviderVersionOverride parses a single -override-provider value.
//
// The expected format is SOURCE=VERSION, where SOURCE is a provider source
// address (for example "hashicorp/aws" or
// "registry.terraform.io/hashicorp/aws") and VERSION is a version constraint
// (for example "5.70.0", ">= 5.0.0", or "~> 5.0").
func ParseProviderVersionOverride(raw string) (ProviderVersionOverride, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	ret := ProviderVersionOverride{Raw: raw}

	source, version, ok := splitOverrideProvider(raw)
	if !ok {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Invalid -override-provider value",
			fmt.Sprintf("The -override-provider option expects SOURCE=VERSION, for example -override-provider=hashicorp/aws=5.70.0. %q is not valid.", raw),
		))
		return ret, diags
	}

	addr, addrDiags := addrs.ParseProviderSourceString(source)
	diags = diags.Append(addrDiags)
	if addrDiags.HasErrors() {
		return ret, diags
	}

	constraint, err := providerreqs.ParseVersionConstraints(version)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Invalid -override-provider version constraint",
			fmt.Sprintf("The version constraint %q for provider %s is invalid: %s.", version, addr.ForDisplay(), err),
		))
		return ret, diags
	}

	ret.Addr = addr
	ret.Version = version
	ret.Constraint = constraint
	return ret, diags
}

// ParseProviderVersionOverrides parses a list of -override-provider values.
// Later values for the same provider replace earlier ones.
func ParseProviderVersionOverrides(raws []string) ([]ProviderVersionOverride, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	if len(raws) == 0 {
		return nil, diags
	}

	seen := make(map[addrs.Provider]int)
	ret := make([]ProviderVersionOverride, 0, len(raws))
	for _, raw := range raws {
		override, overrideDiags := ParseProviderVersionOverride(raw)
		diags = diags.Append(overrideDiags)
		if overrideDiags.HasErrors() {
			continue
		}
		if ix, exists := seen[override.Addr]; exists {
			ret[ix] = override
			continue
		}
		seen[override.Addr] = len(ret)
		ret = append(ret, override)
	}
	return ret, diags
}

func splitOverrideProvider(raw string) (source, version string, ok bool) {
	source, version, found := strings.Cut(raw, "=")
	if !found {
		return "", "", false
	}
	source = strings.TrimSpace(source)
	version = strings.TrimSpace(version)
	if source == "" || version == "" {
		return "", "", false
	}
	return source, version, true
}
